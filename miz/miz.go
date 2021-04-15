package miz

import (
	"archive/zip"
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

	// search file for section containing weather information, date info, and
	// time info
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
		switch {
		// calculate wind speed at 8000 meters
		case strings.Contains(line, `["at8000"]`) && !strings.Contains(line, "end of"):
			// use ground speed to estimate speed at 8000 meters
			speed := data.Data[0].Wind.SpeedMPS
			speed = windSpeed(8000, 1.5, speed, 0.04)
			speedStr := fmt.Sprintf("%0.3f", speed)

			// replace line with calculated speed and save result
			lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + speedStr + ","
			fmt.Println("Updated wind:")
			fmt.Println("\tAt 8000 meters:")
			fmt.Println("\t\tSpeed m/s:", speedStr)

			// offset wind direction by [45,90), dcs expects direction wind moves towards
			dir := rand.Intn(45) + 45 + int(data.Data[0].Wind.Degrees+180)
			dir %= 360
			lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(dir) + ","
			fmt.Println("\t\tDirection:", (dir+180)%360)

		// calculate wind speed at 2000 meters
		case strings.Contains(line, `["at2000"]`) && !strings.Contains(line, "end of"):
			// use ground speed to estimate speed at 2000 meters
			speed := data.Data[0].Wind.SpeedMPS
			speed = windSpeed(2000, 1.5, speed, 0.04)
			speedStr := fmt.Sprintf("%0.3f", speed)

			// replace line with calculated speed and save result
			lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + speedStr + ","
			fmt.Println("\tAt 2000 meters:")
			fmt.Println("\t\tSpeed m/s:", speedStr)

			// offset wind direction by [0,45), dcs expects direction wind moves towards
			dir := rand.Intn(45) + int(data.Data[0].Wind.Degrees+180)
			dir %= 360
			lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(dir) + ","
			fmt.Println("\t\tDirection:", (dir+180)%360)

		// update wind speed for ground level
		case strings.Contains(line, `["atGround"]`) && !strings.Contains(line, "end of"):
			// use metar reported data for ground conditions
			speed := data.Data[0].Wind.SpeedMPS
			speedStr := fmt.Sprintf("%0.3f", speed)

			lines[i+startWeather+2] = "\t\t\t\t[\"speed\"] = " + speedStr + ","
			fmt.Println("\tAt 0 meters:")
			fmt.Println("\t\tSpeed m/s:", speedStr)

			dir := int(data.Data[0].Wind.Degrees + 180)
			dir %= 360
			lines[i+startWeather+3] = "\t\t\t\t[\"dir\"] = " + strconv.Itoa(dir) + ","
			fmt.Println("\t\tDirection:", (dir+180)%360)

		// update turbulence using gust data from metar
		case strings.Contains(line, `["groundTurbulence"]`):
			// check for gusting and use to set ground turbulence
			gust := data.Data[0].Wind.GustMPS
			gustStr := fmt.Sprintf("%0.3f", gust)
			lines[i+startWeather] = "\t\t[\"groundTurbulence\"] = " + gustStr + ","
			fmt.Println("Turbulence:", gust)

		// update temperature
		case strings.Contains(line, `["temperature"]`):
			temp := int(data.Data[0].Temperature.Celsius)
			lines[i+startWeather] = "\t\t\t[\"temperature\"] = " + strconv.Itoa(temp) + ","
			fmt.Println("Temperature Celsius:", temp)

		// update QNH
		case strings.Contains(line, `["qnh"]`):
			// dcs expects QNH in mmHg = inHg * 25.4
			qnh := int(data.Data[0].Barometer.Hg*25.4 + .5)
			lines[i+startWeather] = "\t\t[\"qnh\"] = " + strconv.Itoa(qnh) + ","
			fmt.Println("QNH mmHg:", qnh)

		// update fog visibility
		case strings.Contains(line, `["fog"]`) && !strings.Contains(line, "end of"):
			// thickness is assumed to be 100 meters for now since this is not
			// reported in the metar
			fog := checkFog(data)
			if fog > 0 {
				lines[i+startWeather+2] = "\t\t\t[\"thickness\"] = 100,"
				lines[i+startWeather+3] = "\t\t\t[\"visibility\"] = " + strconv.Itoa(fog) + ","
			}
			fmt.Println("Fog Visibility meters:", fog)

		// enable or disable fog
		case strings.Contains(line, `["enable_fog"]`):
			// enable fog if checkFog returns a valid visibility
			if checkFog(data) > 0 {
				lines[i+startWeather] = "\t\t[\"enable_fog\"] = true,"
				fmt.Println("Fog Enabled:", true)
			} else {
				lines[i+startWeather] = "\t\t[\"enable_fog\"] = false,"
				fmt.Println("Fog Enabled:", false)
			}

		// update dust visibility
		case strings.Contains(line, `["dust_density"]`):
			dust := checkDust(data)
			if dust > 0 {
				lines[i+startWeather] = "\t\t[\"dust_density\"] = " + strconv.Itoa(dust) + ","
				fmt.Println("Dust Visibility meters:", dust)
			}

		// enable or disable dust
		case strings.Contains(line, `["enable_dust"]`):
			if checkDust(data) > 0 {
				lines[i+startWeather] = "\t\t[\"enable_dust\"] = true,"
				fmt.Println("Dust Enabled:", true)
			} else {
				lines[i+startWeather] = "\t\t[\"enable_dust\"] = false,"
				fmt.Println("Dust Enabled:", false)
			}

		// update clouds
		case strings.Contains(line, `["clouds"]`) && !strings.Contains(line, "end of"):
			preset, base := checkClouds(data)

			lines[i+startWeather+2] = "\t\t\t[\"thickness\"] = 200,"
			lines[i+startWeather+3] = "\t\t\t[\"density\"] = 0,"
			if preset == "" {
				lines[i+startWeather+4] = ""
			} else {
				lines[i+startWeather+4] = "\t\t\t[\"preset\"] = " + preset + ","
			}
			lines[i+startWeather+5] = "\t\t\t[\"base\"] = " + strconv.Itoa(base) + ","
			lines[i+startWeather+6] = "\t\t\t[\"iprecptns\"] = 0,"
			fmt.Println("Clouds:")
			fmt.Println("\tPreset:", preset)
			fmt.Println("\tBase meters:", base)
		}
	}

	// update mission date
	year, month, day := parseDate(data)
	lines[startDate+2] = "\t\t[\"Day\"] = " + strconv.Itoa(day) + ","
	lines[startDate+3] = "\t\t[\"Year\"] = " + strconv.Itoa(year) + ","
	lines[startDate+4] = "\t\t[\"Month\"] = " + strconv.Itoa(month) + ","
	fmt.Println("year:", year)
	fmt.Println("month:", month)
	fmt.Println("day:", day)

	// update mission time
	t := parseTime()
	lines[startTime] = "\t[\"start_time\"] = " + strconv.Itoa(t) + ","
	fmt.Println("time:", t)

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

// parseTime returns system time in seconds with offset defined in config file
func parseTime() int {
	// parse config file for parameters
	config := util.ParseConfig()

	// get system time in second
	t := time.Now()
	t = t.Add(config.HourOffset * time.Hour)

	return ((t.Hour()*60)+t.Minute())*60 + t.Second()
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

	if kind == "CAVOK" {
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
	src := util.ParseConfig().InputFile
	fmt.Println("Source file:", src)
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

	dest := util.ParseConfig().OutputFile
	outFile, err := os.Create(dest)
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
