package miz

import (
	"archive/zip"
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

func Update(data weather.WeatherData) error {
	l := lua.NewState()
	if err := l.DoFile("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("Error parsing mission file: %v", err)
	}

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

	if err := l.DoString(writemission); err != nil {
		return fmt.Errorf("Error loading write mission file: %v", err)
	}

	if err := os.Remove("mission_unpacked/mission"); err != nil {
		return fmt.Errorf("Error removing mission: %v", err)
	}

	if err := l.DoString(`writeMission(mission, "mission_unpacked/mission")`); err != nil {
		return fmt.Errorf("Error writing mission file: %v", err)
	}

	return nil
}

func updateWeather(data weather.WeatherData, l *lua.LState) error {
	speed8000 := windSpeed(8000, data)
	speed2000 := windSpeed(2000, data)
	speedGround := windSpeed(1, data)

	dir8000 := (rand.Intn(45) + 45 + int(data.Data[0].Wind.Degrees+180)) % 360
	dir2000 := (rand.Intn(45) + int(data.Data[0].Wind.Degrees+180)) % 360
	dirGround := int(data.Data[0].Wind.Degrees)

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

	temp := data.Data[0].Temperature.Celsius

	if err := l.DoString(
		fmt.Sprintf("mission.weather.season.temperature = %0.3f\n", temp),
	); err != nil {
		return fmt.Errorf("Error updating temperature: %v", err)
	}

	log.Printf("Temperature Celsius: %0.3f\n", temp)

	// qnh is in mmHg = inHg * 25.4
	qnh := int(data.Data[0].Barometer.Hg*25.4 + 0.5)

	if err := l.DoString(
		fmt.Sprintf("mission.weather.qnh = %d\n", qnh),
	); err != nil {
		return fmt.Errorf("Error updating QNH: %v", err)
	}

	log.Printf("QNH mmHg: %d\n", qnh)

	fog := checkFog(data)

	if fog > 0 {
		if err := l.DoString(
			// assume fog thickness 100 since not reported in metar
			fmt.Sprintf(
				"mission.weather.enable_fog = true"+
					"mission.weather.fog.thickness = 100\n"+
					"mission.weather.fog.visibility = %d\n",
				fog,
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
			"\tVisibility meters: %d\n"+
			"\tEnabled: %t\n",
		fog, fog > 0,
	)

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

	preset, base := checkClouds(data)
	weather.SelectedPreset = preset

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

func updateTime(data weather.WeatherData, l *lua.LState) error {
	year, month, day, err := parseDate(data)
	if err != nil {
		return fmt.Errorf("Error parsing date: %v", err)
	}

	t := parseTime()

	if err := l.DoString(
		fmt.Sprintf(
			"mission.date.Year = %d\n"+
				"mission.date.Month = %d\n"+
				"mission.date.Day = %d\n"+
				"mission.start_time = %d\n",
			year, month, day, t,
		),
	); err != nil {
		return fmt.Errorf("Error updating time: %v", err)
	}

	log.Printf(
		"Time:\n"+
			"\tYear: %d\n"+
			"\tMonth: %d\n"+
			"\tDay: %d\n"+
			"\tStart Time: %d\n",
		year, month, day, t,
	)

	return nil
}

// returns extrapolated wind speed at given height using power law
// https://en.wikipedia.org/wiki/Wind_profile_power_law
// targHeight should be provided in meters MSL
func windSpeed(targHeight float64, data weather.WeatherData) float64 {
	// default to 9 meters for reference height if elevation is below that
	refHeight := math.Max(9, data.Data[0].Elevation.Meters)

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
		if condition.Code == "RA" || condition.Code == "SN" {
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

	for _, cloud := range data.Data[0].Clouds {
		if cloud.Code == "BKN" || cloud.Code == "OVC" {
			ceiling = true
		}
	}

	for _, cloud := range data.Data[0].Clouds {
		if (cloud.Code == "FEW" || cloud.Code == "SCT") && ceiling {
			continue
		}
		preset, base = selectPreset(cloud.Code, int(cloud.Meters))
	}

	precip := checkPrecip(data)
	if precip > 0 {
		preset, base = selectPreset("OVC+RA", base)
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
		if preset.MinBase <= base && base <= preset.MaxBase {
			validPresets = append(validPresets, preset)
		}
	}
	nValids := len(validPresets)
	if nValids > 0 {
		preset := validPresets[rand.Intn(nValids)]
		return preset.Name, base
	} else {
		print(kind)
		nPresets := len(weather.CloudPresets[kind])
		preset := weather.CloudPresets[kind][rand.Intn(nPresets)]
		base = int(util.Clamp(float64(base), float64(preset.MinBase), float64(preset.MaxBase)))
		return preset.Name, base
	}
}

// checkFog looks for either misty or foggy conditions and returns and integer
// representing dcs visiblity scale
func checkFog(data weather.WeatherData) int {
	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "FG" || condition.Code == "BR" {
			return int(data.Data[0].Visibility.MetersFloat)
		}
	}
	return 0
}

// checkFog looks for either misty or foggy conditions and returns and integer
// representing dcs visiblity scale
func checkDust(data weather.WeatherData) int {
	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "HZ" || condition.Code == "DU" ||
			condition.Code == "SA" || condition.Code == "PO" ||
			condition.Code == "DS" || condition.Code == "SS" {
			return int(data.Data[0].Visibility.MetersFloat)
		}
	}
	return 0
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file to dest, taken from https://golangcode.com/unzip-files-in-go/
func Unzip() ([]string, error) {
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
	log.Println("unzipped:\n\t" + strings.Join(filenames, "\n\t"))

	return filenames, nil
}

// Zip takes the unpacked mission and recreates the mission file
// taken from https://golangcode.com/create-zip-files-in-go/
func Zip() error {
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
		log.Println(basePath + file.Name())
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

			// Recurse
			newBase := basePath + file.Name() + "/"
			log.Println("Recursing and Adding SubDir: " + file.Name())
			log.Println("Recursing and Adding SubDir: " + newBase)

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
	log.Println("Removed mission_unpacked")
}

// lua function to write table to file
const writemission = `
do
	function writeMission(t, f)

		local function writeMissionHelper(obj, cnt)

			local cnt = cnt or 0

			if type(obj) == "table" then

				io.write("\n", string.rep("\t", cnt), "{\n")
				cnt = cnt + 1

				for k,v in pairs(obj) do

					if type(k) == "string" then
						io.write(string.rep("\t",cnt), '["'..k..'"]', ' = ')
					end

					if type(k) == "number" then
						io.write(string.rep("\t",cnt), "["..k.."]", " = ")
					end

					writeMissionHelper(v, cnt)
					io.write(",\n")
				end

				cnt = cnt-1
				io.write(string.rep("\t", cnt), "}")

			elseif type(obj) == "string" then
				io.write(string.format("%q", obj))

			else
				io.write(tostring(obj))
			end
		end

		io.output(f)
		io.write("mission =")
		writeMissionHelper(t)
		io.output(io.stdout)
	end
end
`
