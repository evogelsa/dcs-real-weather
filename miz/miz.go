package miz

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

func Update(data weather.WeatherData) {
	// open mission lua file
	input, err := ioutil.ReadFile("mission_unpacked/mission")
	util.Must(err)

	// create string array of mission file separated by lines
	lines := strings.Split(string(input), "\n")

	// search file for section containing weather information
	var startWeather int
	var endWeather int
	var startDate int
	var startTime int
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

	// separate just weather lines from file to decrease risk of finding duplicates
	weatherLines := lines[startWeather:endWeather]

	for i, line := range weatherLines {
		if strings.Contains(line, `["at8000"]`) && !strings.Contains(line, "end of") {
			// use ground speed to estimate speed at 8000 feet
			speed := int(data.Data[0].Wind.SpeedKTS)
			speed = int(windSpeed(8000, 100, float64(speed), 0.05))
			lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + strconv.Itoa(speed) + ","

			// offset wind direction by [45,90), dcs expects direction wind moves towards
			dir := rand.Intn(45) + 45 + int(data.Data[0].Wind.Degrees+180)
			dir %= 360
			lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(dir) + ","
		} else if strings.Contains(line, `["at2000"]`) && !strings.Contains(line, "end of") {
			// estimate speed at 2000 feet
			speed := int(data.Data[0].Wind.SpeedKTS)
			speed = int(windSpeed(2000, 100, float64(speed), 0.05))
			lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + strconv.Itoa(speed) + ","

			// offset wind direction by [0,45), dcs expects direction wind moves towards
			dir := rand.Intn(45) + int(data.Data[0].Wind.Degrees+180)
			dir %= 360
			lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(dir) + ","
		} else if strings.Contains(line, `["atGround"]`) && !strings.Contains(line, "end of") {
			// use metar reported data for ground conditions
			speed := int(data.Data[0].Wind.SpeedKTS)
			lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + strconv.Itoa(speed) + ","

			dir := int(data.Data[0].Wind.Degrees + 180)
			dir %= 360
			lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(dir) + ","
		} else if strings.Contains(line, `["temperature"]`) {
			// replace temperature with metar report
			lines[i+startWeather] = "\t\t\t[\"temperature\"] = " +
				strconv.Itoa(int(data.Data[0].Temperature.Celsius)) + ","
		} else if strings.Contains(line, `["qnh"]`) {
			// dcs has linear scale from inHg to "units" given by factor of 25.4
			// round to nearest int
			lines[i+startWeather] = "\t\t[\"qnh\"] = " +
				strconv.Itoa(int(data.Data[0].Barometer.Hg*25.4+.5)) + ","
		} else if strings.Contains(line, `["fog"]`) && !strings.Contains(line, "end of") {
			// thickness is assumed to be 300 for now
			if checkFog(data) > 0 {
				lines[i+startWeather+2] = "\t\t\t[\"thickness\"] = 300,"
				lines[i+startWeather+3] = "\t\t\t[\"visibility\"] = " +
					strconv.Itoa(checkFog(data)) + ","
			}
		} else if strings.Contains(line, `["enable_fog"]`) {
			// enable fog if checkFog returns a valid visibility
			if checkFog(data) > 0 {
				lines[i+startWeather] = "\t\t[\"enable_fog\"] = true" + ","
			} else {
				lines[i+startWeather] = "\t\t[\"enable_fog\"] = false" + ","
			}
		} else if strings.Contains(line, `["clouds"]`) && !strings.Contains(line, "end of") {
			thickness, density, base := checkClouds(data)
			precip := checkPrecip(data)
			lines[i+startWeather+2] = "\t\t\t[\"thickness\"] = " + strconv.Itoa(thickness) + ","
			lines[i+startWeather+3] = "\t\t\t[\"density\"] = " + strconv.Itoa(density) + ","
			lines[i+startWeather+4] = "\t\t\t[\"base\"] = " + strconv.Itoa(base) + ","
			lines[i+startWeather+5] = "\t\t\t[\"iprecptns\"] = " + strconv.Itoa(precip) + ","
		}
	}

	// update mission date
	year, month, day := parseDate(data)
	lines[startDate+2] = "\t\t[\"Day\"] = " + strconv.Itoa(day) + ","
	lines[startDate+3] = "\t\t[\"Year\"] = " + strconv.Itoa(year) + ","
	lines[startDate+4] = "\t\t[\"Month\"] = " + strconv.Itoa(month) + ","

	// update mission time
	t := parseTime()
	lines[startTime] = "\t[\"start_time\"] = " + strconv.Itoa(t) + ","

	// overwrite file with newly changed mission
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile("mission_unpacked/mission", []byte(output), 0644)
	util.Must(err)
}

// returns extrapolated wind speed at given height using log law
// https://websites.pmc.ucsc.edu/~jnoble/wind/extrap/
func windSpeed(targHeight, refHeight, refSpeed, roughness float64) float64 {
	return refSpeed * ((math.Log(targHeight / roughness)) / (math.Log(refHeight / roughness)))
}

// sysTime returns system time in seconds
func sysTime() int {
	t := time.Now()
	return ((t.Hour()*60)+t.Minute())*60 + t.Second()
}

// parseTime returns system time in seconds with offset defined in config file
func parseTime() int {
	// parse config file for parameters
	var config util.Configuration
	file, err := os.Open("config.json")
	util.Must(err)
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	util.Must(err)

	// get system time in second
	t := sysTime()
	// add hour offset from configuration file
	t += config.TimeOffset * 60 * 60
	t %= 24 * 60 * 60

	return t
}

// parseDate returns year, month, day from METAR observed
func parseDate(data weather.WeatherData) (int, int, int) {
	year, err := strconv.Atoi(data.Data[0].Observed[0:4])
	util.Must(err)
	month, err := strconv.Atoi(data.Data[0].Observed[5:7])
	util.Must(err)
	day, err := strconv.Atoi(data.Data[0].Observed[8:10])
	util.Must(err)
	return year, month, day
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
func checkClouds(data weather.WeatherData) (int, int, int) {
	for _, cloud := range data.Data[0].Clouds {
		if cloud.Feet >= 984 {
			var thickness int // 0 - 2000
			var density int   // 0 - 10
			var base int = int(cloud.Feet * .305)
			switch cloud.Code {
			case "FEW":
				thickness = rand.Intn(500)
				density = rand.Intn(3)
			case "SCT":
				thickness = rand.Intn(1000)
				density = rand.Intn(3) + 3
			case "BKN":
				thickness = rand.Intn(1500)
				density = rand.Intn(3) + 6
			case "OVC":
				thickness = rand.Intn(2000)
				density = rand.Intn(2) + 9 // 9 or 10
			}
			if checkPrecip(data) == 2 {
				density = 10
			}
			return thickness, density, base
		}
	}
	// return no clouds
	return 2000, 0, 5000
}

// checkFog looks for either misty or foggy conditions and returns and integer
// representing dcs visiblity scale
func checkFog(data weather.WeatherData) int {
	for _, condition := range data.Data[0].Conditions {
		if condition.Code == "FG" || condition.Code == "BR" {
			// dcs visiblity scales with feet by factor of .305
			return int(data.Data[0].Visibility.MilesFloat*5280*.305 + .5)
		}
	}
	return 0
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file to dest, taken from https://golangcode.com/unzip-files-in-go/
func Unzip() ([]string, error) {

	src := MISSION_NAME
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
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
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

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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
	fmt.Println("unzipped:\n" + strings.Join(filenames, "\n"))

	return filenames, nil
}

// Zip takes the unpacked mission and recreates the mission file
// taken from https://golangcode.com/create-zip-files-in-go/
func Zip() {
	baseFolder := "mission_unpacked/"

	outFile, err := os.Create("realweather.miz")
	util.Must(err)
	defer outFile.Close()

	w := zip.NewWriter(outFile)

	addFiles(w, baseFolder, "")

	err = w.Close()
	if err != nil {
		fmt.Println(err)
	}
}

// addFiles handles adding each file in directory to zip archive
// taken from https://golangcode.com/create-zip-files-in-go/
func addFiles(w *zip.Writer, basePath, baseInZip string) {
	files, err := ioutil.ReadDir(basePath)
	util.Must(err)

	for _, file := range files {
		fmt.Println(basePath + file.Name())
		if !file.IsDir() {
			dat, err := ioutil.ReadFile(basePath + file.Name())
			util.Must(err)

			// Add some files to the archive.
			f, err := w.Create(baseInZip + file.Name())
			util.Must(err)

			_, err = f.Write(dat)
			util.Must(err)
		} else if file.IsDir() {

			// Recurse
			newBase := basePath + file.Name() + "/"
			fmt.Println("Recursing and Adding SubDir: " + file.Name())
			fmt.Println("Recursing and Adding SubDir: " + newBase)

			addFiles(w, newBase, baseInZip+file.Name()+"/")
		}
	}
}

// Clean will remove the unpacked mission from directory
func Clean() {
	directory := "mission_unpacked/"
	os.RemoveAll(directory)
	fmt.Println("Removed mission_unpacked")
}
