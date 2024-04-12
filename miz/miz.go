package miz

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/evogelsa/DCS-real-weather/config"
	"github.com/evogelsa/DCS-real-weather/util"
	"github.com/evogelsa/DCS-real-weather/weather"

	lua "github.com/yuin/gopher-lua"
)

//go:embed datadumper.lua
var dataDumper string

var l *lua.LState

func init() {
	l = lua.NewState(lua.Options{
		RegistrySize:     1024,
		RegistryMaxSize:  1024 * 1024,
		RegistryGrowStep: 1024,
	})

	if err := l.DoString(dataDumper); err != nil {
		log.Fatalf("Error loading data dumper: %v", err)
	}
}

// Update applies weather and time updates to the unpacked mission file
func Update(data weather.WeatherData) error {
	log.Println("Loading mission into Lua VM...")

	// load mission file into lua vm
	if err := l.DoFile("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("Error parsing mission file: %v", err)
	}

	log.Println("Loaded mission into Lua VM")
	log.Println("Updating mission...")

	// update weather if enabled
	if config.Get().Options.UpdateWeather {
		if err := updateWeather(data, l); err != nil {
			return fmt.Errorf("Error updating weather: %v", err)
		}
	}

	// update time if enabled
	if config.Get().Options.UpdateTime {
		if err := updateTime(data, l); err != nil {
			return fmt.Errorf("Error updating time: %v", err)
		}
	}

	log.Println("Updated mission")
	log.Println("Writing new mission file...")

	// remove and write new mission file by dumping lua state

	if err := os.Remove("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("Error removing mission: %v", err)
	}

	if err := l.DoString(`rw_miz = DataDumper(mission, "mission", false, 0)`); err != nil {
		return fmt.Errorf("Error serializing lua state: %v", err)
	}

	lv := l.GetGlobal("rw_miz")
	if s, ok := lv.(lua.LString); ok {
		os.WriteFile("mission_unpacked/mission", []byte(string(s)), 0666)
	} else {
		return fmt.Errorf("Error dumping serialized state")
	}

	log.Println("Wrote new mission file")

	return nil
}

// UpdateBrief updates the unpacked mission brief with the generated METAR
func UpdateBrief(metar string) error {
	log.Println("Loading mission brief into Lua VM...")

	// load brief into lua vm
	if err := l.DoFile("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("Error loading mission dictionary: %v", err)
	}

	log.Println("Loaded mission brief into Lua VM")
	log.Println("Adding METAR to mission brief...")

	// add whitespace to beginning of metar so its separate from brief
	metar = `\n\n` + metar

	// add metar to bottom of brief
	if err := l.DoString(
		"dictionary.DictKey_descriptionText_1 = " +
			"dictionary.DictKey_descriptionText_1 .. " + `"` + metar + `"`,
	); err != nil {
		return fmt.Errorf("Error updating mission brief: %v", err)
	}

	// update brief by removing old and dumping lua state as new file

	if err := os.Remove("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("Error removing mission dictionary: %v", err)
	}

	if err := l.DoString(`rw_dict = DataDumper(dictionary, "dictionary", false, 0)`); err != nil {
		return fmt.Errorf("Error serializing lua state: %v", err)
	}

	lv := l.GetGlobal("rw_dict")
	if s, ok := lv.(lua.LString); ok {
		os.WriteFile("mission_unpacked/l10n/DEFAULT/dictionary", []byte(string(s)), 0666)
	} else {
		return fmt.Errorf("Error dumping serialized state")
	}

	log.Println("Added METAR to mission brief")

	return nil
}

// updateWeather applies new weather to the given lua state using data
func updateWeather(data weather.WeatherData, l *lua.LState) error {
	if err := updateWind(data, l); err != nil {
		return fmt.Errorf("Error updating weather: %v", err)
	}

	if err := updateTemperature(data, l); err != nil {
		return fmt.Errorf("Error updating weather: %v", err)
	}

	if err := updatePressure(data, l); err != nil {
		return fmt.Errorf("Error updating weather: %v", err)
	}

	if err := updateFog(data, l); err != nil {
		return fmt.Errorf("Error updating weather: %v", err)
	}

	if err := updateDust(data, l); err != nil {
		return fmt.Errorf("Error updating weather: %v", err)
	}

	if err := updateClouds(data, l); err != nil {
		return fmt.Errorf("Error updating weather: %v", err)
	}

	return nil
}

// updateClouds applies cloud data from the given weather to the lua state
func updateClouds(data weather.WeatherData, l *lua.LState) error {
	// determine preset to use and cloud base
	preset, base := checkClouds(data)

	// set state in weather so it can be used for generating METAR
	weather.SelectedPreset = preset
	weather.SelectedBase = base

	// check clouds returns custom, use data to construct custom weather
	if strings.Contains(preset, "CUSTOM") {
		err := handleCustomClouds(data, l, preset, base)
		if err != nil {
			return fmt.Errorf("Error making custom clouds: %v", err)
		}

		return nil
	}

	// add clouds to lua state
	if preset != "" {
		// using a preset
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.clouds.thickness = 200\n"+
					"mission.weather.clouds.density = 0\n"+
					"mission.weather.clouds.preset = %s\n"+
					"mission.weather.clouds.base = %d\n"+
					"mission.weather.clouds.iprecptns = 0\n",
				preset, base,
			),
		); err != nil {
			return fmt.Errorf("Error updating clouds: %v", err)
		}
	} else {
		// using no wx / clear skies
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.clouds.thickness = 200\n"+
					"mission.weather.clouds.density = 0\n"+
					"mission.weather.clouds.preset = nil\n"+
					"mission.weather.clouds.base = %d\n"+
					"mission.weather.clouds.iprecptns = 0\n",
				base,
			),
		); err != nil {
			return fmt.Errorf("Error updating clouds: %v", err)
		}
	}

	log.Printf(
		"Clouds:\n"+
			"\tPreset: %s\n"+
			"\tBase: %d meters (%d feet)\n",
		preset, base, int(float64(base)*weather.MetersToFeet),
	)

	return nil
}

// handleCustomClouds generates legacy weather when no preset capable of matching
// the desired weather
func handleCustomClouds(data weather.WeatherData, l *lua.LState, preset string, base int) error {
	// only one kind possible when using custom
	var thickness int = rand.Intn(1801) + 200 // 200 - 2000
	var density int                           //   0 - 10
	var precip int                            //   0 - 2
	base = util.Clamp(base, 300, 5000)        // 300 - 5000

	//  0 - clear
	//  1 - few
	//  2 - few
	//  3 - few
	//  4 - sct
	//  5 - sct
	//  6 - sct
	//  7 - bkn
	//  8 - bkn
	//  9 - bkn
	// 10 - ovc

	precip = checkPrecip(data)
	var precipStr string
	if precip == 2 {
		// make thunderstorm clouds thicc
		thickness = rand.Intn(501) + 1500 // can be up to 2000
		precipStr = "TS"
	} else if precip == 1 {
		thickness = rand.Intn(1801) + 200 // 200 - 2000
		precipStr = "RA"
	} else {
		precipStr = "None"
	}

	// convert cloud type to layer sky coverage, known as density in DCS
	switch preset[7:] {
	case "OVX":
		fallthrough
	case "OVC":
		density = 10
	case "BKN":
		density = rand.Intn(3) + 7
	case "SCT":
		density = rand.Intn(3) + 4
	case "FEW":
		density = rand.Intn(3) + 1
	default:
		density = 0
	}

	// apply to lua state
	if err := l.DoString(
		fmt.Sprintf(
			"mission.weather.clouds.thickness = %d\n"+
				"mission.weather.clouds.density = %d\n"+
				"mission.weather.clouds.preset = nil\n"+
				"mission.weather.clouds.base = %d\n"+
				"mission.weather.clouds.iprecptns = %d\n",
			thickness,
			density,
			base,
			precip,
		),
	); err != nil {
		return fmt.Errorf("Error updating clouds: %v", err)
	}

	log.Printf(
		"Clouds:\n"+
			"\tPreset: %s\n"+
			"\tBase:      %4d meters (%5d feet)\n"+
			"\tThickness: %4d meters (%5d feet)\n"+
			"\tPrecipitation: %s\n",
		preset,
		base, int(float64(base)*weather.MetersToFeet),
		thickness, int(float64(thickness)*weather.MetersToFeet),
		precipStr,
	)

	return nil
}

// updateDust applies dust to mission if METAR reports dust conditions
func updateDust(data weather.WeatherData, l *lua.LState) error {
	dust := checkDust(data)

	if dust > 0 {
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.dust_density = %d\n"+
					"mission.weather.enable_dust = true\n",
				dust,
			),
		); err != nil {
			return fmt.Errorf("Error updating dust: %v", err)
		}
	} else {
		if err := l.DoString("mission.weather.enable_dust = false"); err != nil {
			return fmt.Errorf("Error updating dust: %v", err)
		}
	}

	log.Printf(
		"Dust:\n"+
			"\tVisibility: %d meters (%d feet)\n"+
			"\tEnabled: %t\n",
		dust, int(float64(dust)*weather.MetersToFeet), dust > 0,
	)

	return nil
}

// updateFog applies fog to mission state
func updateFog(data weather.WeatherData, l *lua.LState) error {
	fogVis, fogThick := checkFog(data)

	if fogVis > 0 {
		if err := l.DoString(
			// assume fog thickness 100 since not reported in metar
			fmt.Sprintf(
				"mission.weather.enable_fog = true\n"+
					"mission.weather.fog.thickness = %d\n"+
					"mission.weather.fog.visibility = %d\n",
				fogThick, fogVis,
			),
		); err != nil {
			return fmt.Errorf("Error updating fog: %v", err)
		}
	} else {
		if err := l.DoString("mission.weather.enable_fog = false"); err != nil {
			return fmt.Errorf("Error updating fog: %v", err)
		}
	}

	log.Printf(
		"Fog:\n"+
			"\tThickness:  %d meters (%d feet)\n"+
			"\tVisibility: %d meters (%d feet)\n"+
			"\tEnabled: %t\n",
		fogThick, int(float64(fogThick)*weather.MetersToFeet),
		fogVis, int(float64(fogThick)*weather.MetersToFeet),
		fogVis > 0,
	)

	return nil
}

// updatePressure applies pressure to mission state
func updatePressure(data weather.WeatherData, l *lua.LState) error {
	// qnh is in mmHg = inHg * 25.4
	qnh := int(data.Data[0].Barometer.Hg*25.4 + 0.5)

	if err := l.DoString(
		fmt.Sprintf("mission.weather.qnh = %d\n", qnh),
	); err != nil {
		return fmt.Errorf("Error updating QNH: %v", err)
	}

	log.Printf("QNH: %d mmHg (%0.2f inHg)\n", qnh, data.Data[0].Barometer.Hg)

	return nil
}

// updateTemperature applies temperature to mission state
func updateTemperature(data weather.WeatherData, l *lua.LState) error {
	temp := data.Data[0].Temperature.Celsius

	if err := l.DoString(
		fmt.Sprintf("mission.weather.season.temperature = %0.3f\n", temp),
	); err != nil {
		return fmt.Errorf("Error updating temperature: %v", err)
	}

	log.Printf("Temperature: %0.1f C (%0.1f F)\n", temp, weather.CelsiusToFahrenheit(temp))

	return nil
}

// updateWind applies reported wind to mission state and also calculates
// and applies winds aloft using wind profile power law. This function also
// applies turbulence/gust data to the mission
func updateWind(data weather.WeatherData, l *lua.LState) error {
	speedGround := windSpeed(1, data)
	speed2000 := windSpeed(2000, data)
	speed8000 := windSpeed(8000, data)

	// cap wind speeds to configured values
	minWind := config.Get().Options.Wind.Minimum
	maxWind := config.Get().Options.Wind.Maximum
	speedGround = util.Clamp(speedGround, minWind, maxWind)
	speed2000 = util.Clamp(speed2000, minWind, maxWind)
	speed8000 = util.Clamp(speed8000, minWind, maxWind)

	// apply wind shift to winds aloft layers
	// this is not really realistic but it adds variety to wind calculation
	dirGround := int(data.Data[0].Wind.Degrees+180) % 360
	dir2000 := (rand.Intn(45) + dirGround) % 360
	dir8000 := (rand.Intn(45) + dir2000) % 360

	// apply to mission state
	if err := l.DoString(
		fmt.Sprintf(
			"mission.weather.wind.at8000.speed = %0.3f\n"+
				"mission.weather.wind.at8000.dir = %d\n"+
				"mission.weather.wind.at2000.speed = %0.3f\n"+
				"mission.weather.wind.at2000.dir = %d\n"+
				"mission.weather.wind.atGround.speed = %0.3f\n"+
				"mission.weather.wind.atGround.dir = %d\n",
			speed8000, dir8000, speed2000, dir2000, speedGround, dirGround,
		),
	); err != nil {
		return fmt.Errorf("Error updating winds: %v", err)
	}

	log.Printf(
		"Winds:\n"+
			"\tAt 8000 meters (26000 ft):\n"+
			"\t\tSpeed: %0.3f m/s (%d kt)\n"+
			"\t\tDirection: %03d\n"+
			"\tAt 2000 meters (6500 ft):\n"+
			"\t\tSpeed: %0.3f m/s (%d kt)\n"+
			"\t\tDirection: %03d\n"+
			"\tAt ground:\n"+
			"\t\tSpeed: %0.3f m/s (%d kt)\n"+
			"\t\tDirection: %03d\n",
		speed8000, int(speed8000*weather.MPSToKT),
		dir8000,
		speed2000, int(speed2000*weather.MPSToKT),
		dir2000,
		speedGround, int(speedGround*weather.MPSToKT),
		dirGround,
	)

	// apply gustiness/turbulence to mission
	gust := data.Data[0].Wind.GustMPS

	if err := l.DoString(
		fmt.Sprintf("mission.weather.groundTurbulence = %0.4f\n", gust),
	); err != nil {
		return fmt.Errorf("Error updating turbulence: %v", err)
	}

	log.Printf("Gusts: %0.3f m/s (%d kt)\n", gust, int(gust*weather.MPSToKT))

	return nil
}

// updateTime applies system time plus/minus configured offset to the mission
func updateTime(data weather.WeatherData, l *lua.LState) error {
	year, month, day, err := parseDate(data)
	if err != nil {
		return fmt.Errorf("Error parsing date: %v", err)
	}

	sec := parseTime()

	if err := l.DoString(
		fmt.Sprintf(
			"mission.date.Year = %d\n"+
				"mission.date.Month = %d\n"+
				"mission.date.Day = %d\n"+
				"mission.start_time = %d\n",
			year, month, day, sec,
		),
	); err != nil {
		return fmt.Errorf("Error updating time: %v", err)
	}

	log.Printf(
		"Time:\n"+
			"\tYear: %d\n"+
			"\tMonth: %d\n"+
			"\tDay: %d\n"+
			"\tStart time: %d (%02d:%02d:%02d)\n",
		year, month, day, sec, sec/3600, (sec%3600)/60, sec%60,
	)

	return nil
}

// returns extrapolated wind speed at given height using power law
// https://en.wikipedia.org/wiki/Wind_profile_power_law
// targHeight should be provided in meters MSL
func windSpeed(targHeight float64, data weather.WeatherData) float64 {
	// default to 9 meters for reference height if elevation is below that
	var refHeight float64
	if config.Get().Options.Wind.FixedReference {
		refHeight = 1
	} else {
		refHeight = math.Max(1, float64(config.Get().METAR.RunwayElevation))
	}

	refSpeed := data.Data[0].Wind.SpeedMPS

	// enforce minimum targheight of 0
	targHeight = math.Max(0, targHeight)

	return refSpeed * math.Pow(
		targHeight/refHeight,
		config.Get().Options.Wind.Stability,
	)
}

// parseTime returns system time in seconds with offset defined in config file
func parseTime() int {
	// get system time in second
	t := time.Now()

	offset, err := time.ParseDuration(config.Get().Options.TimeOffset)
	if err != nil {
		offset = 0
		log.Printf(
			"Could not parse time-offset of %s: %v. Program will default to 0 offset",
			config.Get().Options.TimeOffset,
			err,
		)
	}
	t = t.Add(offset)

	return ((t.Hour()*60)+t.Minute())*60 + t.Second()
}

// parseDate returns year, month, day from METAR observed
func parseDate(data weather.WeatherData) (int, int, int, error) {
	year, err := strconv.Atoi(data.Data[0].Observed[0:4])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Error parsing year from data: %v", err)
	}

	month, err := strconv.Atoi(data.Data[0].Observed[5:7])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Error parsing month from data: %v", err)
	}

	day, err := strconv.Atoi(data.Data[0].Observed[8:10])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Error parsing day from data: %v", err)
	}

	return year, month, day, nil
}

// checkPrecip returns 0 for clear, 1 for rain, and 2 for thunderstorms
func checkPrecip(data weather.WeatherData) int {
	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "RA" || // rain
			condition.Code == "SN" || // snow
			condition.Code == "DZ" || // drizzle
			condition.Code == "SG" || // snow grains
			condition.Code == "GS" || // snow pellets or small hail
			condition.Code == "GR" || // hail
			condition.Code == "PL" || // ice pellets
			condition.Code == "IC" || // ice crystals
			condition.Code == "UP" { // unknown precip
			return 1
		} else if condition.Code[:2] == "TS" {
			return 2
		}
	}
	return 0
}

// checkClouds returns the thickness, density and base of the first cloud
// layer reported in the METAR in feet
func checkClouds(data weather.WeatherData) (string, int) {
	var ceiling bool
	var preset string
	var base int

	base = config.Get().METAR.RunwayElevation

	precip := checkPrecip(data)

	// determine if there is a ceiling
	for _, cloud := range data.Data[0].Clouds {
		if cloud.Code == "BKN" || cloud.Code == "OVC" {
			ceiling = true
		}
	}

	// tracks coverage of fullest layer
	fullestLayer := 0

	// tracks index of layer being used as base
	baseLayer := 0

	// table to convert cloud code to integer
	var codeToVal = map[string]int{
		"OVC": 4,
		"BKN": 3,
		"SCT": 2,
		"FEW": 1,
	}

	// find first layer to be used as base. Prioritizes fullest layer if there
	// is precip, otherwise picks first ceiling if there is ceiling, otherwise
	// picks first layer
	for i, cloud := range data.Data[0].Clouds {
		if precip > 0 {
			if codeToVal[cloud.Code] > fullestLayer {
				fullestLayer = codeToVal[cloud.Code]
				baseLayer = i
			}
		} else if ceiling {
			if cloud.Code == "FEW" || cloud.Code == "SCT" {
				continue
			}
			baseLayer = i
			break
		} else {
			baseLayer = i
			break
		}
	}

	base += int(data.Data[0].Clouds[baseLayer].Meters)
	code := data.Data[0].Clouds[baseLayer].Code

	// updates base with selected in case of fallback to legacy
	preset, base = selectPreset(code, base, precip > 0)

	return preset, base
}

// selectPreset uses the given weather data to best select a preset that is
// suitable. If no suitable preset is found and fallback to no preset is enabled
// then custom weather will be used. If fallback is not enabled but a default
// preset is configured, use that. Otherwise the the preset defaults to clear.
func selectPreset(kind string, base int, precip bool) (string, int) {
	// check for clear skies
	if slices.Contains(weather.ClearCodes(), kind) {
		return "", 0
	}

	// if precip and overcast, then use OVC+RA preset
	// if precip and not ovc but using legacy wx, use legacy wx
	// if precip and not ovc but not using legacy wx, ignore precip

	if precip {
		if kind == "OVC" {
			kind = "OVC+RA"
		} else if config.Get().Options.Clouds.FallbackToNoPreset {
			log.Printf("No suitable weather preset for code=%s and base=%d", kind, base)
			log.Printf("Fallback to no preset is enabled, using custom weather")
			return "CUSTOM " + kind[:3], base
		} else {
			log.Printf("No suitable preset for %s clouds with precip", kind)
			log.Printf("Fallback to no preset is disabled, so precip will be ignored")
		}
	}

	// make a list of possible presets that can be used to match weather
	var validPresets []weather.CloudPreset
	var validPresetsIgnoreBase []weather.CloudPreset
	for _, preset := range weather.CloudPresets[kind] {
		if presetAllowed(preset.Name) {
			if util.Between(base, preset.MinBase, preset.MaxBase) {
				validPresets = append(validPresets, preset)
			} else {
				validPresetsIgnoreBase = append(validPresetsIgnoreBase, preset)
			}
		}
	}

	// randomly select a valid preset
	if len(validPresets) > 0 {
		preset := validPresets[rand.Intn(len(validPresets))]
		return preset.Name, base
	}

	log.Printf("No suitable weather preset for code=%s and base=%d", kind, base)

	// no valid preset found, is use nonpreset weather enabled?
	if config.Get().Options.Clouds.FallbackToNoPreset {
		log.Printf("Fallback to no preset is enabled, using custom weather")
		return "CUSTOM " + kind[:3], base
	}

	log.Printf("Fallback to no preset is disabled. Expanding search to only %s\n", kind)

	// since fallback disabled and no preset available, use any preset that
	// matches the desired cloud type and ignore desired base
	validPresets = validPresetsIgnoreBase

	// still no valid presets? use the configured default preset if there is
	// one, otherwise default to clear
	if len(validPresets) == 0 {
		if config.Get().Options.Clouds.DefaultPreset != "" {
			defaultPreset := config.Get().Options.Clouds.DefaultPreset
			defaultPreset = `"` + defaultPreset + `"`

			log.Printf(
				"No allowed presets for %s. Defaulting to %s.",
				kind,
				defaultPreset,
			)

			base, _ := strconv.Atoi(weather.DecodePreset[defaultPreset][0].Base)

			return defaultPreset, base
		} else {
			log.Printf("No allowed presets for %s. Defaulting to CLR.", kind)
			return "", 0
		}
	}

	// random select a valid preset from the expanded list
	preset := validPresets[rand.Intn(len(validPresets))]
	return preset.Name, rand.Intn(preset.MaxBase-preset.MinBase) + preset.MinBase
}

// presetAllowed checks if a preset is in the disallowed presets inside the
// config file. If the preset is disallowed the func returns false
func presetAllowed(preset string) bool {
	for _, disallowed := range config.Get().Options.Clouds.DisallowedPresets {
		if preset == `"`+disallowed+`"` {
			return false
		}
	}

	return true
}

// checkFog looks for either misty or foggy conditions and returns and integer
// representing dcs visiblity scale
func checkFog(data weather.WeatherData) (visibility, thickness int) {
	if !config.Get().Options.Fog.Enabled {
		return
	}

	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "FG" || condition.Code == "BR" {

			thickness = rand.Intn(
				config.Get().Options.Fog.ThicknessMaximum-
					config.Get().Options.Fog.ThicknessMinimum,
			) + config.Get().Options.Fog.ThicknessMinimum

			visibility = int(util.Clamp(
				data.Data[0].Visibility.MetersFloat,
				config.Get().Options.Fog.VisibilityMinimum,
				config.Get().Options.Fog.VisibilityMaximum,
			))

			return
		}
	}

	return
}

// checkDust looks for dust conditions and returns a number representing
// visibility in meters
func checkDust(data weather.WeatherData) (visibility int) {
	if !config.Get().Options.Dust.Enabled {
		return
	}

	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "HZ" || condition.Code == "DU" ||
			condition.Code == "SA" || condition.Code == "PO" ||
			condition.Code == "DS" || condition.Code == "SS" {

			return int(util.Clamp(
				data.Data[0].Visibility.MetersFloat,
				config.Get().Options.Dust.VisibilityMinimum,
				config.Get().Options.Dust.VisibilityMaximum,
			))
		}
	}
	return 0
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file to dest, taken from https://golangcode.com/unzip-files-in-go/
func Unzip() ([]string, error) {
	log.Println("Unpacking mission file...")

	src := config.Get().Files.InputMission
	log.Println("Source file:", src)
	dest := "mission_unpacked"

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(
			fpath,
			filepath.Clean(dest)+string(os.PathSeparator),
		) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return filenames, err
			}
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(
			fpath,
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			f.Mode(),
		)
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}

	log.Println("Unpacked mission file")
	// log.Println("unzipped:\n\t" + strings.Join(filenames, "\n\t"))

	return filenames, nil
}

// Zip takes the unpacked mission and recreates the mission file
// taken from https://golangcode.com/create-zip-files-in-go/
func Zip() error {
	log.Println("Repacking mission file...")

	baseFolder := "mission_unpacked/"

	dest := config.Get().Files.OutputMission
	outFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("Error creating output file: %v", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	addFiles(w, baseFolder, "")

	err = w.Close()
	if err != nil {
		return fmt.Errorf("Error closing output file: %v", err)
	}

	log.Println("Repacked mission file")

	return nil
}

// addFiles handles adding each file in directory to zip archive
// taken from https://golangcode.com/create-zip-files-in-go/
func addFiles(w *zip.Writer, basePath, baseInZip string) error {
	files, err := os.ReadDir(basePath)
	if err != nil {
		return fmt.Errorf("Error reading directory %v: %v", basePath, err)
	}

	for _, file := range files {
		// log.Println("zipped " + basePath + file.Name())
		if !file.IsDir() {
			dat, err := os.ReadFile(basePath + file.Name())
			if err != nil {
				return fmt.Errorf(
					"Error reading file %v: %v",
					basePath+file.Name(),
					err,
				)
			}

			// Add some files to the archive.
			f, err := w.Create(baseInZip + file.Name())
			if err != nil {
				return fmt.Errorf(
					"Error creating file %v: %v",
					baseInZip+file.Name(),
					err,
				)
			}

			_, err = f.Write(dat)
			if err != nil {
				return fmt.Errorf("Error writing data: %v", err)
			}

		} else if file.IsDir() {
			newBase := basePath + file.Name() + "/"
			err := addFiles(w, newBase, baseInZip+file.Name()+"/")
			if err != nil {
				return fmt.Errorf("Error adding files from %v: %v", baseInZip+file.Name()+"/", err)
			}
		}
	}

	return nil
}

// Clean will remove the unpacked mission from directory
func Clean() {
	directory := "mission_unpacked/"
	os.RemoveAll(directory)
	log.Println("Removed unpacked mission")
}
