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
	"strconv"
	"strings"
	"time"

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

func Update(data weather.WeatherData) error {
	log.Println("Loading mission into Lua VM...")

	if err := l.DoFile("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("Error parsing mission file: %v", err)
	}

	log.Println("Loaded mission into Lua VM")
	log.Println("Updating mission...")

	if util.Config.Options.UpdateWeather {
		if err := updateWeather(data, l); err != nil {
			return fmt.Errorf("Error updating weather: %v", err)
		}
	}

	if util.Config.Options.UpdateTime {
		if err := updateTime(data, l); err != nil {
			return fmt.Errorf("Error updating time: %v", err)
		}
	}

	log.Println("Updated mission")
	log.Println("Writing new mission file...")

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

func UpdateBrief(metar string) error {
	log.Println("Loading mission brief into Lua VM...")

	if err := l.DoFile("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("Error loading mission dictionary: %v", err)
	}

	log.Println("Loaded mission brief into Lua VM")
	log.Println("Adding METAR to mission brief...")

	// add whitespace to beginning of metar so its separate from brief
	metar = "\n\n" + metar

	// add metar to bottom of brief
	if err := l.DoString(
		"dictionary.DictKey_descriptionText_1 = " +
			"dictionary.DictKey_descriptionText_1 .. " + `"` + metar + `"`,
	); err != nil {
		return fmt.Errorf("Error updating mission brief: %v", err)
	}

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

func updateClouds(data weather.WeatherData, l *lua.LState) error {
	preset, base := checkClouds(data)

	// check clouds returns custom, use data to construct custom weather
	if strings.Contains(preset, "CUSTOM") {
		err := handleCustomClouds(data, l, preset, base)
		if err != nil {
			return fmt.Errorf("Error making custom clouds: %v", err)
		}

		weather.SelectedPreset = preset // "CUSTOM + <kind>"
		weather.SelectedBase = base
		return nil
	}

	weather.SelectedPreset = preset
	weather.SelectedBase = base

	if preset != "" {
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
			"\tBase meters: %d\n",
		preset, base,
	)

	return nil
}

func handleCustomClouds(data weather.WeatherData, l *lua.LState, preset string, base int) error {
	// only one kind possible when using custom
	var thickness int = rand.Intn(1801) + 200        // 200 - 2000
	var density int                                  //   0 - 10
	var precip int                                   //   0 - 2
	base = int(util.Clamp(float64(base), 300, 5000)) // 300 - 5000

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
			"\tBase meters: %d\n"+
			"\tThickness meters: %d\n"+
			"\tPrecipitation: %s\n",
		preset,
		base,
		thickness,
		precipStr,
	)

	return nil
}

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
			"\tVisibility meters: %d\n"+
			"\tEnabled: %t\n",
		dust, dust > 0,
	)

	return nil
}

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
			"\tThickness meters: %d\n"+
			"\tVisibility meters: %d\n"+
			"\tEnabled: %t\n",
		fogThick, fogVis, fogVis > 0,
	)

	return nil
}

func updatePressure(data weather.WeatherData, l *lua.LState) error {
	// qnh is in mmHg = inHg * 25.4
	qnh := int(data.Data[0].Barometer.Hg*25.4 + 0.5)

	if err := l.DoString(
		fmt.Sprintf("mission.weather.qnh = %d\n", qnh),
	); err != nil {
		return fmt.Errorf("Error updating QNH: %v", err)
	}

	log.Printf("QNH mmHg: %d\n", qnh)

	return nil
}

func updateTemperature(data weather.WeatherData, l *lua.LState) error {
	temp := data.Data[0].Temperature.Celsius

	if err := l.DoString(
		fmt.Sprintf("mission.weather.season.temperature = %0.3f\n", temp),
	); err != nil {
		return fmt.Errorf("Error updating temperature: %v", err)
	}

	log.Printf("Temperature Celsius: %0.3f\n", temp)

	return nil
}

func updateWind(data weather.WeatherData, l *lua.LState) error {
	speedGround := windSpeed(1, data)
	speed2000 := windSpeed(2000, data)
	speed8000 := windSpeed(8000, data)

	if util.Config.Options.Wind.Maximum >= 0 {
		speedGround = math.Min(speedGround, util.Config.Options.Wind.Maximum)
		speed2000 = math.Min(speed2000, util.Config.Options.Wind.Maximum)
		speed8000 = math.Min(speed8000, util.Config.Options.Wind.Maximum)
	}

	if util.Config.Options.Wind.Minimum >= 0 {
		speedGround = math.Max(speedGround, util.Config.Options.Wind.Minimum)
		speed2000 = math.Max(speed2000, util.Config.Options.Wind.Minimum)
		speed8000 = math.Max(speed8000, util.Config.Options.Wind.Minimum)
	}

	dirGround := int(data.Data[0].Wind.Degrees+180) % 360
	dir2000 := (rand.Intn(45) + dirGround) % 360
	dir8000 := (rand.Intn(45) + dir2000) % 360

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
			"\tAt 8000 meters:\n"+
			"\t\tSpeed m/s: %0.3f\n"+
			"\t\tDirection: %d\n"+
			"\tAt 2000 meters:\n"+
			"\t\tSpeed m/s: %0.3f\n"+
			"\t\tDirection: %d\n"+
			"\tAt 1 meters:\n"+
			"\t\tSpeed m/s: %0.3f\n"+
			"\t\tDirection: %d\n",
		speed8000, dir8000, speed2000, dir2000, speedGround, dirGround,
	)

	gust := data.Data[0].Wind.GustMPS

	if err := l.DoString(
		fmt.Sprintf("mission.weather.groundTurbulence = %0.4f\n", gust),
	); err != nil {
		return fmt.Errorf("Error updating turbulence: %v", err)
	}

	log.Printf("Gusts m/s: %0.3f\n", gust)

	return nil
}

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
	if util.Config.Options.Wind.FixedReference {
		refHeight = 1
	} else {
		refHeight = math.Max(1, data.Data[0].Elevation.Meters)
	}

	refSpeed := data.Data[0].Wind.SpeedMPS

	// enforce minimum targheight of 0
	targHeight = math.Max(0, targHeight)

	return refSpeed * math.Pow(
		targHeight/refHeight,
		util.Config.Options.Wind.Stability,
	)
}

// parseTime returns system time in seconds with offset defined in config file
func parseTime() int {
	// get system time in second
	t := time.Now()

	offset, err := time.ParseDuration(util.Config.Options.TimeOffset)
	if err != nil {
		offset = 0
		log.Printf(
			"Could not parse time-offset of %s: %v. Program will default to 0 offset",
			util.Config.Options.TimeOffset,
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
		} else if condition.Code == "TS" {
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

	precip := checkPrecip(data)
	if precip > 0 {
		preset, base = selectPreset("OVC+RA", base)
		return preset, base
	}

	for _, cloud := range data.Data[0].Clouds {
		if cloud.Code == "BKN" || cloud.Code == "OVC" {
			ceiling = true
		}
	}

	// prioritizes selecting a present based on ceiling rather than base
	for _, cloud := range data.Data[0].Clouds {
		if (cloud.Code == "FEW" || cloud.Code == "SCT") && ceiling {
			continue
		}
		preset, base = selectPreset(cloud.Code, int(cloud.Meters))
		break
	}

	return preset, base
}

func selectPreset(kind string, base int) (string, int) {
	var validPresets []weather.CloudPreset

	if kind == "CAVOK" || kind == "CLR" || kind == "SKC" || kind == "NSC" ||
		kind == "NCD" {
		return "", 0
	}

	for _, preset := range weather.CloudPresets[kind] {
		if presetAllowed(preset.Name) && preset.MinBase <= base && base <= preset.MaxBase {
			validPresets = append(validPresets, preset)
		}
	}

	if len(validPresets) > 0 {
		preset := validPresets[rand.Intn(len(validPresets))]
		return preset.Name, base
	}

	log.Printf("No suitable weather preset for code=%s and base=%d", kind, base)

	// no valid preset found, is use nonpreset weather enabled?
	if util.Config.Options.FallbackToNoPreset {
		log.Printf("Fallback to no preset is enabled, using custom weather")
		return "CUSTOM " + kind[:3], base
	}

	log.Printf("Fallback to no preset is disabled. Expanding search to only %s\n", kind)

	for _, preset := range weather.CloudPresets[kind] {
		if presetAllowed(preset.Name) {
			validPresets = append(validPresets, preset)
		}
	}

	if len(validPresets) == 0 {
		log.Printf("No allowed presets for %s. Defaulting to CLR.", kind)
		return "", 0
	}

	preset := validPresets[rand.Intn(len(validPresets))]
	return preset.Name, rand.Intn(preset.MaxBase-preset.MinBase) + preset.MinBase
}

// presetAllowed checks if a preset is in the disallowed presets inside the
// config file. If the preset is disallowed the func returns false
func presetAllowed(preset string) bool {
	for _, disallowed := range util.Config.Options.Clouds.DisallowedPresets {
		if preset == `"`+disallowed+`"` {
			return false
		}
	}

	return true
}

// checkFog looks for either misty or foggy conditions and returns and integer
// representing dcs visiblity scale
func checkFog(data weather.WeatherData) (visibility, thickness int) {
	if !util.Config.Options.Fog.Enabled {
		return
	}

	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "FG" || condition.Code == "BR" {

			if util.Config.Options.Fog.ThicknessMaximum > 1000 {
				log.Println("Fog maximum thickness is set above max of 1000; defaulting to 1000")
				util.Config.Options.Fog.ThicknessMaximum = 1000
			}

			if util.Config.Options.Fog.ThicknessMinimum < 0 {
				log.Println("Fog minimum thickness is set below min of 0; defaulting to 0")
				util.Config.Options.Fog.ThicknessMinimum = 0
			}

			thickness = rand.Intn(
				util.Config.Options.Fog.ThicknessMaximum-
					util.Config.Options.Fog.ThicknessMinimum,
			) + util.Config.Options.Fog.ThicknessMinimum

			if util.Config.Options.Fog.VisibilityMaximum > 6000 {
				log.Println("Fog maximum visibility is set above max of 6000; defaulting to 6000")
				util.Config.Options.Fog.VisibilityMaximum = 6000
			}

			if util.Config.Options.Fog.VisibilityMinimum < 0 {
				log.Println("Fog minimum visibility is set below min of 0; defaulting to 0")
				util.Config.Options.Fog.VisibilityMinimum = 0
			}

			visibility = int(util.Clamp(
				data.Data[0].Visibility.MetersFloat,
				float64(util.Config.Options.Fog.VisibilityMinimum),
				float64(util.Config.Options.Fog.VisibilityMaximum),
			))

			return
		}
	}

	return
}

// checkDust looks for dust conditions and returns a number representing
// visibility in meters
func checkDust(data weather.WeatherData) (visibility int) {
	if !util.Config.Options.Dust.Enabled {
		return
	}

	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "HZ" || condition.Code == "DU" ||
			condition.Code == "SA" || condition.Code == "PO" ||
			condition.Code == "DS" || condition.Code == "SS" {

			if util.Config.Options.Dust.VisibilityMinimum < 300 {
				log.Println("Dust visibility minimum is set below min of 300; defaulting to 300")
				util.Config.Options.Dust.VisibilityMinimum = 300
			}

			if util.Config.Options.Dust.VisibilityMaximum > 3000 {
				log.Println("Dust visibility maximum is set above max of 3000; defaulting to 3000")
				util.Config.Options.Dust.VisibilityMaximum = 3000
			}

			return int(util.Clamp(
				data.Data[0].Visibility.MetersFloat,
				float64(util.Config.Options.Dust.VisibilityMinimum),
				float64(util.Config.Options.Dust.VisibilityMaximum),
			))
		}
	}
	return 0
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file to dest, taken from https://golangcode.com/unzip-files-in-go/
func Unzip() ([]string, error) {
	log.Println("Unpacking mission file...")

	src := util.Config.Files.InputMission
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

	dest := util.Config.Files.OutputMission
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
