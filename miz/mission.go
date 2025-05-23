package miz

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/evogelsa/DCS-real-weather/v2/config"
	"github.com/evogelsa/DCS-real-weather/v2/logger"
	"github.com/evogelsa/DCS-real-weather/v2/util"
	"github.com/evogelsa/DCS-real-weather/v2/weather"
)

type precipitation int

const (
	precipNone precipitation = iota
	precipSome
	precipStorm
)

// UpdateMission applies weather and time updates to the unpacked mission file
func UpdateMission(data *weather.WeatherData, windsAloft weather.WindsAloft) error {
	logger.Infoln("loading mission into Lua VM...")

	// load mission file into lua vm
	if err := l.DoFile("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("error parsing mission file: %v", err)
	}

	logger.Infoln("loaded mission into Lua VM")
	logger.Infoln("updating mission...")

	// update weather if enabled
	if config.Get().Options.Weather.Enable {
		// remove extra weather data and add copy for output
		data.Data = []weather.Data{data.Data[0], data.Data[0]}
		if err := updateWeather(data, windsAloft, l); err != nil {
			return fmt.Errorf("error updating weather: %v", err)
		}
	}

	// update time if enabled
	if config.Get().Options.Time.Enable {
		if err := updateTime(data, l); err != nil {
			return fmt.Errorf("error updating time: %v", err)
		}
	}

	// update date if enabled
	if config.Get().Options.Date.Enable {
		if err := updateDate(data, l); err != nil {
			return fmt.Errorf("error updating date: %v", err)
		}
	}

	logger.Infoln("updated mission")
	logger.Infoln("writing new mission file...")

	// remove and write new mission file by dumping lua state

	if err := os.Remove("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("error removing mission: %v", err)
	}

	lv := l.GetGlobal("mission")
	if tbl, ok := lv.(*lua.LTable); ok {
		s := serializeTable(tbl, 0)
		s = "mission = " + s
		os.WriteFile("mission_unpacked/mission", []byte(s), 0666)
	} else {
		return fmt.Errorf("error dumping serialized state")
	}

	logger.Infoln("wrote new mission file")

	return nil
}

// updateWeather applies new weather to the given lua state using data
func updateWeather(data *weather.WeatherData, windsAloft weather.WindsAloft, l *lua.LState) error {
	if config.Get().Options.Weather.Wind.Enable {
		if config.Get().API.OpenMeteo.Enable {
			if err := updateWind(data, windsAloft, l); err != nil {
				return fmt.Errorf("error updating wind: %v", err)
			}
		} else {
			if err := updateWindLegacy(data, l); err != nil {
				return fmt.Errorf("error updating wind: %v", err)
			}
		}
	}

	if config.Get().Options.Weather.Temperature.Enable {
		if err := updateTemperature(data, l); err != nil {
			return fmt.Errorf("error updating temperature: %v", err)
		}
	}

	if config.Get().Options.Weather.Pressure.Enable {
		if err := updatePressure(data, l); err != nil {
			return fmt.Errorf("error updating pressure: %v", err)
		}
	}

	if config.Get().Options.Weather.Fog.Enable {
		if err := updateFog(data, l); err != nil {
			return fmt.Errorf("error updating fog: %v", err)
		}
	}

	if config.Get().Options.Weather.Dust.Enable {
		if err := updateDust(data, l); err != nil {
			return fmt.Errorf("error updating dust: %v", err)
		}
	}

	if config.Get().Options.Weather.Clouds.Enable {
		if err := updateClouds(data, l); err != nil {
			return fmt.Errorf("error updating clouds: %v", err)
		}
	}

	return nil
}

// updateClouds applies cloud data from the given weather to the lua state
func updateClouds(data *weather.WeatherData, l *lua.LState) error {
	// determine preset to use and cloud base
	preset, base := checkClouds(data)

	// set state in weather so it can be used for generating METAR
	weather.SelectedPreset = preset
	weather.SelectedBase = base - int(config.Get().Options.Weather.RunwayElevation+0.5)

	// check clouds returns custom, use data to construct custom weather
	if strings.Contains(preset, "CUSTOM") {
		err := handleCustomClouds(data, l, preset, base)
		if err != nil {
			return fmt.Errorf("error making custom clouds: %v", err)
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
			return fmt.Errorf("error updating clouds: %v", err)
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
			return fmt.Errorf("error updating clouds: %v", err)
		}
	}

	logger.Infow(
		"clouds:",
		"preset", preset,
		"base-meters", base,
		"base-feet", int(float64(base)*weather.MetersToFeet),
	)

	return nil
}

// handleCustomClouds generates legacy weather when no preset capable of matching
// the desired weather
func handleCustomClouds(data *weather.WeatherData, l *lua.LState, preset string, base int) error {
	// only one kind possible when using custom
	var thickness int = rand.Intn(1801) + 200 // 200 - 2000
	var density int                           //   0 - 10
	precip := precipNone                      //   0 - 2
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

	if config.Get().Options.Weather.Clouds.Custom.AllowPrecipitation {
		precip = checkPrecip(data)
	}

	var precipStr string
	if precip == precipStorm {
		// make thunderstorm clouds thicc
		thickness = rand.Intn(501) + 1500 // can be up to 2000
		precipStr = "TS"
	} else if precip == precipSome {
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

	density = util.Clamp(
		density,
		config.Get().Options.Weather.Clouds.Custom.DensityMinimum,
		config.Get().Options.Weather.Clouds.Custom.DensityMaximum,
	)

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
		return fmt.Errorf("error updating clouds: %v", err)
	}

	logger.Infow(
		"clouds:",
		"preset", preset,
		"base-meters", base,
		"base-feet", int(float64(base)*weather.MetersToFeet),
		"thickness-meters", thickness,
		"thickness-feet", int(float64(thickness)*weather.MetersToFeet),
		"precipitation", precipStr,
	)

	return nil
}

// updateDust applies dust to mission if METAR reports dust conditions
func updateDust(data *weather.WeatherData, l *lua.LState) error {
	dust := checkDust(data)

	if dust > 0 {
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.dust_density = %d\n"+
					"mission.weather.enable_dust = true\n",
				dust,
			),
		); err != nil {
			return fmt.Errorf("error updating dust: %v", err)
		}
	} else {
		if err := l.DoString("mission.weather.enable_dust = false"); err != nil {
			return fmt.Errorf("error updating dust: %v", err)
		}
	}

	logger.Infow(
		"dust:",
		"visibility-meters", dust,
		"visibility-feet", int(float64(dust)*weather.MetersToFeet),
		"enabled", dust > 0,
	)

	return nil
}

// updateFog applies fog to mission state
func updateFog(data *weather.WeatherData, l *lua.LState) error {
	fogVis, fogThick := checkFog(data)

	if fogVis <= 0 {
		if err := l.DoString(
			"mission.weather.enable_fog = false\n" +
				"mission.weather.fog2 = nil\n",
		); err != nil {
			return fmt.Errorf("error updating fog: %v", err)
		}
		logger.Infow(
			"fog:",
			"thickness-meters", fogThick,
			"thickness-feet", int(float64(fogThick)*weather.MetersToFeet),
			"visibility-meters", fogVis,
			"visibility-feet", int(float64(fogVis)*weather.MetersToFeet),
			"mode", "manual",
			"enabled", false,
		)
		return nil
	}

	switch weather.Fog(config.Get().Options.Weather.Fog.Mode) {
	case weather.FogLegacy:
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.enable_fog = true\n"+
					"mission.weather.fog.thickness = %d\n"+
					"mission.weather.fog.visibility = %d\n"+
					"mission.weather.fog2 = nil",
				fogThick,
				fogVis,
			),
		); err != nil {
			return fmt.Errorf("error updating fog: %v", err)
		}

		logger.Infow(
			"fog:",
			"thickness-meters", fogThick,
			"thickness-feet", int(float64(fogThick)*weather.MetersToFeet),
			"visibility-meters", fogVis,
			"visibility-feet", int(float64(fogVis)*weather.MetersToFeet),
			"mode", "legacy",
			"enabled", true,
		)

	case weather.FogManual:
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.enable_fog = false\n"+
					"mission.weather.fog2 = { }\n"+
					"mission.weather.fog2.mode = 4\n"+
					"mission.weather.fog2.manual = { { thickness = %d, time = 0, visibility = %d } }",
				fogThick,
				fogVis,
			),
		); err != nil {
			return fmt.Errorf("error updating fog: %v", err)
		}

		logger.Infow(
			"fog:",
			"thickness-meters", fogThick,
			"thickness-feet", int(float64(fogThick)*weather.MetersToFeet),
			"visibility-meters", fogVis,
			"visibility-feet", int(float64(fogVis)*weather.MetersToFeet),
			"mode", "manual",
			"enabled", true,
		)

	default:
		logger.Errorf(
			"unknown fog option \"%s\"",
			string(config.Get().Options.Weather.Fog.Mode),
		)
		logger.Warnln("using fog mode \"auto\"")
		fallthrough
	case weather.FogAuto:
		if err := l.DoString(
			fmt.Sprintf(
				"mission.weather.enable_fog = false\n" +
					"mission.weather.fog2 = { }\n" +
					"mission.weather.fog2.mode = 2\n",
			),
		); err != nil {
			return fmt.Errorf("error updating fog: %v", err)
		}

		logger.Infow(
			"fog:",
			"thickness-meters", "auto",
			"thickness-feet", "auto",
			"visibility-meters", "auto",
			"visibility-feet", "auto",
			"mode", "auto",
			"enabled", true,
		)
	}

	return nil
}

// updatePressure applies pressure to mission state
func updatePressure(data *weather.WeatherData, l *lua.LState) error {
	// convert qnh to qff
	qnh := data.Data[0].Barometer.Hg * weather.InHgToHPa
	elevation := float64(config.Get().Options.Weather.RunwayElevation)
	temperature := data.Data[0].Temperature.Celsius
	latitude := data.Data[0].Station.Geometry.Coordinates[1]
	qff := weather.QNHToQFF(qnh, elevation, temperature, latitude)

	// convert to mmHg
	qff *= weather.HPaToInHg * weather.InHgToMMHg

	if err := l.DoString(
		fmt.Sprintf("mission.weather.qnh = %d\n", int(qff+0.5)),
	); err != nil {
		return fmt.Errorf("error updating pressure: %v", err)
	}

	logger.Infow(
		"pressure:",
		"hPa", qnh,
		"inHg", data.Data[0].Barometer.Hg,
	)

	return nil
}

// updateTemperature applies temperature to mission state
func updateTemperature(data *weather.WeatherData, l *lua.LState) error {
	temp := data.Data[0].Temperature.Celsius

	if err := l.DoString(
		fmt.Sprintf("mission.weather.season.temperature = %0.3f\n", temp),
	); err != nil {
		return fmt.Errorf("error updating temperature: %v", err)
	}

	logger.Infow(
		"temperature:",
		"celsius", temp,
		"fahrenheit", weather.CelsiusToFahrenheit(temp),
	)

	return nil
}

// updateWind uses open meteo data to get winds aloft data then applies this to
// the mission state
func updateWind(data *weather.WeatherData, windsAloft weather.WindsAloft, l *lua.LState) error {
	speedGround := windSpeed(1, data)

	// cap wind speeds to configured values
	minWind := config.Get().Options.Weather.Wind.Minimum
	maxWind := config.Get().Options.Weather.Wind.Maximum
	speedGround = util.Clamp(speedGround, minWind, maxWind)
	speed2000 := util.Clamp(windsAloft.WindSpeed1900, minWind, maxWind)
	speed8000 := util.Clamp(windsAloft.WindSpeed7200, minWind, maxWind)

	// set speed to data out
	data.Data[1].Wind.SpeedMPS = speedGround

	dirGround := int(data.Data[0].Wind.Degrees)
	dir2000 := windsAloft.WindDirection1900
	dir8000 := windsAloft.WindDirection7200

	// clamp wind directions to configured values
	minDir := config.Get().Options.Weather.Wind.DirectionMinimum
	maxDir := config.Get().Options.Weather.Wind.DirectionMaximum
	dirGround = util.Clamp(dirGround, minDir, maxDir)
	dir2000 = util.Clamp(dir2000, minDir, maxDir)
	dir8000 = util.Clamp(dir8000, minDir, maxDir)

	// DCS expects winds TO direction, but standard everywhere else is
	// winds from direction. convert here with + 180 % 360
	dirGround = (dirGround + 180) % 360
	dir2000 = (dir2000 + 180) % 360
	dir8000 = (dir8000 + 180) % 360

	// set direction to data out
	data.Data[1].Wind.Degrees = float64((dirGround + 180) % 360)

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
		return fmt.Errorf("error updating winds: %v", err)
	}

	logger.Infow(
		"winds:",
		"8000-meters", map[string]any{
			"mps": speed8000,
			"kt":  speed8000 * weather.MPSToKT,
			"dir": (dir8000 + 180) % 360,
		},
		"2000-meters", map[string]any{
			"mps": speed2000,
			"kt":  speed2000 * weather.MPSToKT,
			"dir": (dir2000 + 180) % 360,
		},
		"ground", map[string]any{
			"mps": speedGround,
			"kt":  speedGround * weather.MPSToKT,
			"dir": (dirGround + 180) % 360,
		},
	)

	// apply gustiness/turbulence to mission
	gust := data.Data[0].Wind.GustMPS
	minGust := config.Get().Options.Weather.Wind.GustMinimum
	maxGust := config.Get().Options.Weather.Wind.GustMaximum
	gust = util.Clamp(gust, minGust, maxGust)

	// update data out
	data.Data[1].Wind.GustMPS = gust

	// convert to ED gust units (whatever those are?)
	gust *= weather.MPSToEDUnits

	if err := l.DoString(
		fmt.Sprintf("mission.weather.groundTurbulence = %0.4f\n", gust),
	); err != nil {
		return fmt.Errorf("error updating turbulence: %v", err)
	}

	logger.Infow(
		"gusts:",
		"mps", gust,
		"kt", gust*weather.MPSToKT,
	)

	return nil
}

// updateWindLegacy applies reported wind to mission state and also calculates
// and applies winds aloft using wind profile power law. This function also
// applies turbulence/gust data to the mission
func updateWindLegacy(data *weather.WeatherData, l *lua.LState) error {
	speedGround := windSpeed(1, data)
	speed2000 := windSpeed(2000, data)
	speed8000 := windSpeed(8000, data)

	// cap wind speeds to configured values
	minWind := config.Get().Options.Weather.Wind.Minimum
	maxWind := config.Get().Options.Weather.Wind.Maximum
	speedGround = util.Clamp(speedGround, minWind, maxWind)
	speed2000 = util.Clamp(speed2000, minWind, maxWind)
	speed8000 = util.Clamp(speed8000, minWind, maxWind)

	// update data out
	data.Data[1].Wind.SpeedMPS = speedGround

	// apply wind shift to winds aloft layers
	// this is not really realistic but it adds variety to wind calculation
	dirGround := int(data.Data[0].Wind.Degrees)
	dir2000 := rand.Intn(45) + dirGround
	dir8000 := rand.Intn(45) + dir2000

	// clamp wind directions to configured values
	minDir := config.Get().Options.Weather.Wind.DirectionMinimum
	maxDir := config.Get().Options.Weather.Wind.DirectionMaximum
	dirGround = util.Clamp(dirGround, minDir, maxDir)
	dir2000 = util.Clamp(dir2000, minDir, maxDir)
	dir8000 = util.Clamp(dir8000, minDir, maxDir)

	// DCS expects winds TO direction, but standard everywhere else is
	// winds from direction. convert here with + 180 % 360
	dirGround = (dirGround + 180) % 360
	dir2000 = (dir2000 + 180) % 360
	dir8000 = (dir8000 + 180) % 360

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
		return fmt.Errorf("error updating winds: %v", err)
	}

	logger.Infow(
		"winds:",
		"8000-meters", map[string]any{
			"mps": speed8000,
			"kt":  speed8000 * weather.MPSToKT,
			"dir": (dir8000 + 180) % 360,
		},
		"2000-meters", map[string]any{
			"mps": speed2000,
			"kt":  speed2000 * weather.MPSToKT,
			"dir": (dir2000 + 180) % 360,
		},
		"ground", map[string]any{
			"mps": speedGround,
			"kt":  speedGround * weather.MPSToKT,
			"dir": (dirGround + 180) % 360,
		},
	)

	// apply gustiness/turbulence to mission
	gust := data.Data[0].Wind.GustMPS
	minGust := config.Get().Options.Weather.Wind.GustMinimum
	maxGust := config.Get().Options.Weather.Wind.GustMaximum
	gust = util.Clamp(gust, minGust, maxGust)

	// update data out
	data.Data[1].Wind.GustMPS = gust

	if err := l.DoString(
		fmt.Sprintf("mission.weather.groundTurbulence = %0.4f\n", gust),
	); err != nil {
		return fmt.Errorf("error updating turbulence: %v", err)
	}

	logger.Infow(
		"gusts:",
		"mps", gust,
		"kt", gust*weather.MPSToKT,
	)

	return nil
}

// updateTime applies time plus/minus configured offset to the mission
func updateTime(data *weather.WeatherData, l *lua.LState) error {
	var t time.Time
	var err error
	if config.Get().Options.Time.SystemTime {
		t = time.Now()
	} else {
		t, err = time.Parse("2006-01-02T15:04:05", data.Data[0].Observed)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z", data.Data[0].Observed)
			if err != nil {
				logger.Errorf("error parsing METAR time: %v", err)
				logger.Warnln("using system time as fallback")
				t = time.Now()
			}
		}
	}

	offset, err := time.ParseDuration(config.Get().Options.Time.Offset)
	if err != nil {
		logger.Errorf("could not parse time-offset of %s: %v", config.Get().Options.Time.Offset, err)
		logger.Warnln("using default offset of 0")
		offset = 0
	}
	t = t.Add(offset)

	seconds := ((t.Hour()*60)+t.Minute())*60 + t.Second()

	if err := l.DoString(
		fmt.Sprintf(
			"mission.start_time = %d\n",
			seconds,
		),
	); err != nil {
		return fmt.Errorf("error updating time: %v", err)
	}

	logger.Infow(
		"time:",
		"seconds", seconds,
		"clock", fmt.Sprintf(
			"%02d:%02d:%02d",
			t.Hour(), t.Minute(), t.Second(),
		),
	)

	return nil
}

// updateDate applies date plus/minus configured offset to the mission
func updateDate(data *weather.WeatherData, l *lua.LState) error {
	var t time.Time
	var err error
	if config.Get().Options.Date.SystemDate {
		t = time.Now()
	} else {
		t, err = time.Parse("2006-01-02T15:04:05", data.Data[0].Observed)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05Z", data.Data[0].Observed)
			if err != nil {
				logger.Errorf("error parsing METAR date: %v", err)
				logger.Warnln("using system date as fallback")
				t = time.Now()
			}
		}
	}

	offset, err := util.ParseDateDuration(config.Get().Options.Date.Offset)
	if err != nil {
		logger.Errorf("could not parse time-offset of %s: %v", config.Get().Options.Date.Offset, err)
		logger.Warnln("using default offset of 0")
		offset = 0
	}
	t = t.Add(offset)

	if err := l.DoString(
		fmt.Sprintf(
			"mission.date.Year = %d\n"+
				"mission.date.Month = %d\n"+
				"mission.date.Day = %d\n",
			t.Year(), t.Month(), t.Day(),
		),
	); err != nil {
		return fmt.Errorf("error updating date: %v", err)
	}

	logger.Infow(
		"date:",
		"year", t.Year(),
		"month", t.Month(),
		"day", t.Day(),
	)

	return nil
}

// returns extrapolated wind speed at given height using power law
// https://en.wikipedia.org/wiki/Wind_profile_power_law
// targHeight should be provided in meters MSL
func windSpeed(targHeight float64, data *weather.WeatherData) float64 {
	// default to 9 meters for reference height if elevation is below that
	var refHeight float64
	if config.Get().Options.Weather.Wind.FixedReference {
		refHeight = 1
	} else {
		refHeight = math.Max(1, float64(config.Get().Options.Weather.RunwayElevation))
	}

	refSpeed := data.Data[0].Wind.SpeedMPS

	// enforce minimum targheight of 0
	targHeight = math.Max(0, targHeight)

	return refSpeed * math.Pow(
		targHeight/refHeight,
		config.Get().Options.Weather.Wind.Stability,
	)
}

// checkPrecip returns 0 for clear, 1 for rain, and 2 for thunderstorms
func checkPrecip(data *weather.WeatherData) precipitation {
	for _, condition := range data.Data[0].Conditions {
		if slices.Contains(weather.PrecipCodes(), condition.Code) {
			return precipSome
		} else if slices.Contains(weather.StormCodes(), condition.Code[:2]) {
			return precipStorm
		}
	}
	return precipNone
}

// checkClouds returns the thickness, density and base of the first cloud
// layer reported in the METAR in meters
func checkClouds(data *weather.WeatherData) (string, int) {
	var ceiling bool
	var preset string
	var base int

	// if no clouds then assume clear
	if len(data.Data[0].Clouds) == 0 {
		return "", 0
	}

	base = int(config.Get().Options.Weather.RunwayElevation + 0.5)

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
	codeToVal := map[string]int{
		"OVC": 4,
		"BKN": 3,
		"SCT": 2,
		"FEW": 1,
	}

	// find first layer to be used as base. Prioritizes fullest layer if there
	// is precip, otherwise picks first ceiling if there is ceiling, otherwise
	// picks first layer
	for i, cloud := range data.Data[0].Clouds {
		if precip > precipNone {
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

	// clamp base between configured min and max
	base = util.Clamp(
		base,
		config.Get().Options.Weather.Clouds.Base.Minimum,
		config.Get().Options.Weather.Clouds.Base.Maximum,
	)

	// updates base with selected in case of fallback to legacy
	preset, base = selectPreset(code, base, precip > precipNone)

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
	// if precip and broken, then use BKN+RA preset
	// if precip and scattered, then use SCT+RA preset
	// if precip and not ovc/bkn/sct but using legacy wx, use legacy wx
	// if precip and not ovc/bkn/sct but not using legacy wx, ignore precip

	if precip {
		if kind == "OVC" {
			kind = "OVC+RA"
		} else if kind == "BKN" {
			kind = "BKN+RA"
		} else if kind == "SCT" {
			kind = "SCT+RA"
		} else if config.Get().Options.Weather.Clouds.Custom.Enable {
			logger.Warnf("no suitable weather preset for code=%s and base=%d", kind, base)
			logger.Infoln("custom clouds are enabled, using custom weather")
			return "CUSTOM " + kind[:3], base
		} else {
			logger.Warnf("no suitable preset for %s clouds with precip", kind)
			logger.Warnln("fallback to no preset is disabled, so precip will be ignored")
		}
	}

	// make a list of possible presets that can be used to match weather
	var validPresets []weather.CloudPreset
	var validPresetsIgnoreBase []weather.CloudPreset
	for _, preset := range weather.CloudPresets[kind] {
		if presetAllowed(preset.Name) {
			if util.Between(base, preset.MinBase, preset.MaxBase) {
				validPresets = append(validPresets, preset)
			} else if preset.MinBase < int(config.Get().Options.Weather.Clouds.Base.Maximum) &&
				preset.MaxBase > int(config.Get().Options.Weather.Clouds.Base.Minimum) {
				// we also construct a list of presets that don't have a cloud
				// base range that allow for matching the METAR base; however,
				// these presets must still be constrained by the configured
				// min and max base. These are used if no preset match is made
				// and the search must be expanded (and deviate from the METAR)
				validPresetsIgnoreBase = append(validPresetsIgnoreBase, preset)
			}
		}
	}

	// randomly select a valid preset
	if len(validPresets) > 0 {
		preset := validPresets[rand.Intn(len(validPresets))]
		return preset.Name, base
	}

	logger.Warnf("no suitable weather preset for code=%s and base=%d", kind, base)

	// no valid preset found, is use nonpreset weather enabled?
	if config.Get().Options.Weather.Clouds.Custom.Enable {
		logger.Infoln("custom clouds are enabled, using custom weather")
		return "CUSTOM " + kind[:3], base
	}

	logger.Infof("fallback to no preset is disabled, expanding search to only %s\n", kind)

	// since fallback disabled and no preset available, expand valid presets to
	// include those that matches the desired cloud type and ignore desired base
	// (but base still falls within configured limits)
	validPresets = append(validPresets, validPresetsIgnoreBase...)

	// still no valid presets? use the configured default preset if there is
	// one, otherwise default to clear
	if len(validPresets) == 0 {
		if config.Get().Options.Weather.Clouds.Presets.Default != "" {
			defaultPreset := config.Get().Options.Weather.Clouds.Presets.Default
			defaultPreset = `"` + defaultPreset + `"`

			logger.Warnf("no allowed presets for %s", kind)
			logger.Warnf("defaulting to %s", defaultPreset)

			// get base in hundreds of feet
			base, _ := strconv.Atoi(weather.DecodePreset[defaultPreset][0].Base)

			// convert to feet
			base *= 100

			// convert to meters
			base = int(float64(base)*weather.FeetToMeters + 0.5)

			// clamp base between desired min and max base. DCS should clamp
			// the value to the correct range for the preset, so the configured
			// min and max base may be ignored if this happens. the user should
			// have been warned of this possibility during the config validation
			base = util.Clamp(
				base,
				config.Get().Options.Weather.Clouds.Base.Minimum,
				config.Get().Options.Weather.Clouds.Base.Maximum,
			)

			return defaultPreset, base
		} else {
			logger.Warnf("no allowed presets for %s", kind)
			logger.Warnln("defaulting to CLR")
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
	for _, disallowed := range config.Get().Options.Weather.Clouds.Presets.Disallowed {
		if preset == `"`+disallowed+`"` {
			return false
		}
	}

	return true
}

// checkFog looks for either misty or foggy conditions and returns and integer
// representing dcs visiblity scale
func checkFog(data *weather.WeatherData) (visibility, thickness int) {
	for _, condition := range data.Data[0].Conditions {
		if slices.Contains(weather.FogCodes(), condition.Code) {
			thickness = rand.Intn(
				int(config.Get().Options.Weather.Fog.ThicknessMaximum+0.5)-
					int(config.Get().Options.Weather.Fog.ThicknessMinimum+0.5),
			) + int(config.Get().Options.Weather.Fog.ThicknessMinimum+0.5)

			visibility = int(util.Clamp(
				data.Data[0].Visibility.MetersFloat,
				config.Get().Options.Weather.Fog.VisibilityMinimum,
				config.Get().Options.Weather.Fog.VisibilityMaximum,
			))

			return
		}
	}

	return
}

// checkDust looks for dust conditions and returns a number representing
// visibility in meters
func checkDust(data *weather.WeatherData) (visibility int) {
	for _, condition := range data.Data[0].Conditions {
		if slices.Contains(weather.DustCodes(), condition.Code) {
			return int(util.Clamp(
				data.Data[0].Visibility.MetersFloat,
				config.Get().Options.Weather.Dust.VisibilityMinimum,
				config.Get().Options.Weather.Dust.VisibilityMaximum,
			))
		}
	}
	return 0
}
