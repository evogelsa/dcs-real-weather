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
)

const MISSION_NAME string = "mission.miz"

func Update(data weather.WeatherData) error {
	// open mission lua file
	input, err := os.ReadFile("mission_unpacked/mission")
	if err != nil {
		return fmt.Errorf("Error reading unpacked mission file: %v", err)
	}

	// create string array of mission file separated by lines
	lines := strings.Split(string(input), "\n")

	// search file for section containing weather information, date info, and
	// time info
	var startWeather, endWeather, startDate, startTime int = searchLines(lines)

	if util.Config.UpdateWeather {
		// separate just weather lines from file to decrease risk of finding duplicates
		weatherLines := lines[startWeather:endWeather]

		for i, line := range weatherLines {
			switch {
			// calculate wind speed at 8000 meters
			case strings.Contains(line, `["at8000"]`) && !strings.Contains(line, "end of"):
				// use reported wind speeds to estimate winds at 8000 meters MSL
				speed := windSpeed(8000, data)
				speedStr := fmt.Sprintf("%0.3f", speed)

				// replace line with calculated speed and save result
				lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + speedStr + ","
				log.Println("Updated wind:")
				log.Println("\tAt 8000 meters:")
				log.Println("\t\tSpeed m/s:", speedStr)

				// offset wind direction by [45,90), dcs expects direction wind moves towards
				dir := rand.Intn(45) + 45 + int(data.Data[0].Wind.Degrees+180)
				dir %= 360
				lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(
					dir,
				) + ","
				log.Println("\t\tDirection:", (dir+180)%360)

				// calculate wind speed at 2000 meters
			case strings.Contains(line, `["at2000"]`) && !strings.Contains(line, "end of"):
				// use reported wind speeds to estimate winds at 2000 meters MSL
				speed := windSpeed(2000, data)
				speedStr := fmt.Sprintf("%0.3f", speed)

				// replace line with calculated speed and save result
				lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + speedStr + ","
				log.Println("\tAt 2000 meters:")
				log.Println("\t\tSpeed m/s:", speedStr)

				// offset wind direction by [0,45), dcs expects direction wind moves towards
				dir := rand.Intn(45) + int(data.Data[0].Wind.Degrees+180)
				dir %= 360
				lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(
					dir,
				) + ","
				log.Println("\t\tDirection:", (dir+180)%360)

				// update wind speed for ground level
			case strings.Contains(line, `["atGround"]`) && !strings.Contains(line, "end of"):
				// use wind speeds estimated at 1m agl for ground winds
				speed := windSpeed(1, data)
				speedStr := fmt.Sprintf("%0.3f", speed)

				lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + speedStr + ","
				log.Println("\tAt 0 meters:")
				log.Println("\t\tSpeed m/s:", speedStr)

				dir := int(data.Data[0].Wind.Degrees + 180)
				dir %= 360
				lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(
					dir,
				) + ","
				log.Println("\t\tDirection:", (dir+180)%360)

				// update turbulence using gust data from metar
			case strings.Contains(line, `["groundTurbulence"]`):
				// check for gusting and use to set ground turbulence
				gust := data.Data[0].Wind.GustMPS
				gustStr := fmt.Sprintf("%0.3f", gust)
				lines[i+startWeather] = "\t\t[\"groundTurbulence\"] = " + gustStr + ","
				log.Println("Turbulence:", gust)

				// update temperature
			case strings.Contains(line, `["temperature"]`):
				temp := int(data.Data[0].Temperature.Celsius)
				lines[i+startWeather] = "\t\t\t[\"temperature\"] = " + strconv.Itoa(
					temp,
				) + ","
				log.Println("Temperature Celsius:", temp)

				// update QNH
			case strings.Contains(line, `["qnh"]`):
				// dcs expects QNH in mmHg = inHg * 25.4
				qnh := int(data.Data[0].Barometer.Hg*25.4 + .5)
				lines[i+startWeather] = "\t\t[\"qnh\"] = " + strconv.Itoa(
					qnh,
				) + ","
				log.Println("QNH mmHg:", qnh)

				// update fog visibility
			case strings.Contains(line, `["fog"]`) && !strings.Contains(line, "end of"):
				// thickness is assumed to be 100 meters for now since this is not
				// reported in the metar
				fog := checkFog(data)
				if fog > 0 {
					lines[i+startWeather+2] = "\t\t\t[\"thickness\"] = 100,"
					lines[i+startWeather+3] = "\t\t\t[\"visibility\"] = " + strconv.Itoa(
						fog,
					) + ","
				}
				log.Println("Fog Visibility meters:", fog)

				// enable or disable fog
			case strings.Contains(line, `["enable_fog"]`):
				// enable fog if checkFog returns a valid visibility
				if checkFog(data) > 0 {
					lines[i+startWeather] = "\t\t[\"enable_fog\"] = true,"
					log.Println("Fog Enabled:", true)
				} else {
					lines[i+startWeather] = "\t\t[\"enable_fog\"] = false,"
					log.Println("Fog Enabled:", false)
				}

				// update dust visibility
			case strings.Contains(line, `["dust_density"]`):
				dust := checkDust(data)
				if dust > 0 {
					lines[i+startWeather] = "\t\t[\"dust_density\"] = " + strconv.Itoa(
						dust,
					) + ","
					log.Println("Dust Visibility meters:", dust)
				}

				// enable or disable dust
			case strings.Contains(line, `["enable_dust"]`):
				if checkDust(data) > 0 {
					lines[i+startWeather] = "\t\t[\"enable_dust\"] = true,"
					log.Println("Dust Enabled:", true)
				} else {
					lines[i+startWeather] = "\t\t[\"enable_dust\"] = false,"
					log.Println("Dust Enabled:", false)
				}

				// update clouds
			case strings.Contains(line, `["clouds"]`) && !strings.Contains(line, "end of"):
				preset, base := checkClouds(data)
				weather.SelectedPreset = preset

				lines[i+startWeather+2] = "\t\t\t[\"thickness\"] = 200,"
				lines[i+startWeather+3] = "\t\t\t[\"density\"] = 0,"

				// if miz did not already contain a preset, add space for one
				if !strings.Contains(lines[i+startWeather+4], `["preset"]`) {
					lines = append(
						lines[:i+startWeather+4+1],
						lines[i+startWeather+4:]...,
					)

					// expanded lines by one so need to update indexes
					startWeather, endWeather, startDate, startTime = searchLines(lines)
				}

				if preset == "" {
					lines[i+startWeather+4] = ""
				} else {
					lines[i+startWeather+4] = "\t\t\t[\"preset\"] = " + preset + ","
				}

				lines[i+startWeather+5] = "\t\t\t[\"base\"] = " + strconv.Itoa(
					base,
				) + ","
				lines[i+startWeather+6] = "\t\t\t[\"iprecptns\"] = 0,"
				log.Println("Clouds:")
				log.Println("\tPreset:", preset)
				log.Println("\tBase meters:", base)
			}
		}
	}

	// update mission date and time if update-time true in config
	if util.Config.UpdateTime {
		// update date
		year, month, day, err := parseDate(data)
		if err != nil {
			return fmt.Errorf("Error parsing date: %v", err)
		}

		lines[startDate+2] = "\t\t[\"Day\"] = " + strconv.Itoa(day) + ","
		lines[startDate+3] = "\t\t[\"Year\"] = " + strconv.Itoa(year) + ","
		lines[startDate+4] = "\t\t[\"Month\"] = " + strconv.Itoa(month) + ","
		log.Println("year:", year)
		log.Println("month:", month)
		log.Println("day:", day)

		// update time
		t := parseTime()
		lines[startTime] = "\t[\"start_time\"] = " + strconv.Itoa(t) + ","
		log.Println("time:", t)
	}

	// overwrite file with newly changed mission
	output := strings.Join(lines, "\n")
	err = os.WriteFile("mission_unpacked/mission", []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("Error writing to unpacked mission file: %v", err)
	}

	return nil
}

// returns indexes of various important lines in the file
func searchLines(
	lines []string,
) (startWeather, endWeather, startDate, startTime int) {
	for i, line := range lines {
		if strings.Contains(line, `["weather"] =`) {
			startWeather = i
		} else if strings.Contains(line, `}, -- end of ["weather"]`) {
			endWeather = i
		} else if strings.Contains(line, `["date"] =`) {
			startDate = i
		} else if len(line) > 20 {
			if strings.Contains(line[:20], "[\"start_time\"] =") {
				startTime = i
			}
		}
	}
	return
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

	return refSpeed * math.Pow(targHeight/refHeight, util.Config.Stability)
}

// parseTime returns system time in seconds with offset defined in config file
func parseTime() int {
	// get system time in second
	t := time.Now()
	t = t.Add(util.Config.HourOffset * time.Hour)

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
	src := util.Config.InputFile
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

	dest := util.Config.OutputFile
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
