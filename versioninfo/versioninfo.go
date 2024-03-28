package versioninfo

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

//go:generate sh -c "printf %s $(git describe --tags) > git.txt"
//go:embed git.txt
var gitInfo string

const versionInfo = `{
	"FixedFileInfo": {
	  "FileVersion": {
		"Major": %d,
		"Minor": %d,
		"Patch": %d,
		"Build": %d
	  },
	  "ProductVersion": {
		"Major": %d,
		"Minor": %d,
		"Patch": %d,
		"Build": %d
	  }
	},
	"StringFileInfo": {
		"Comments": "",
		"CompanyName": "github.com/evogelsa",
		"FileDescription": "DCS Real Weather Updater",
		"FileVersion": "%d.%d.%d",
		"InternalName": "DCS Real Weather",
		"LegalCopyright": "Copyright 2020 evogelsa",
		"OriginalFilename": "realweather.exe",
		"ProductName": "DCS Real Weather",
		"ProductVersion": "%d.%d.%d.%d"
	},
	"VarFileInfo": {
	  "Translation": {
		  "LangID": "0409",
		  "CharsetID": "0"
	  }
	},
	"IconPath": "versioninfo/icon.ico",
	"ManifestPath": ""
}
`

var re = regexp.MustCompile(`v(?P<Major>\d+)\.(?P<Minor>\d+)\.(?P<Patch>\d+)-?(?P<Pre>(?:alpha)|(?:beta)|(?:rc\d*))?-?(?P<CommitNum>\d*)?.*(?:-g(?P<Commit>\w*))?`)

var (
	Major     int
	Minor     int
	Patch     int
	Pre       string
	CommitNum int
	Commit    string
)

func init() {
	match := re.FindStringSubmatch(gitInfo)
	v := make(map[string]string)

	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			v[name] = match[i]
		}
	}

	Major, _ = strconv.Atoi(v["Major"])
	Minor, _ = strconv.Atoi(v["Minor"])
	Patch, _ = strconv.Atoi(v["Patch"])
	Pre = v["Pre"]
	CommitNum, _ = strconv.Atoi(v["CommitNum"])
	Commit = v["Commit"]

	os.WriteFile(
		"versioninfo/versioninfo.json",
		[]byte(fmt.Sprintf(
			versionInfo,
			Major,
			Minor,
			Patch,
			CommitNum,
			Major,
			Minor,
			Patch,
			CommitNum,
			Major,
			Minor,
			Patch,
			Major,
			Minor,
			Patch,
			CommitNum,
		)),
		os.ModePerm,
	)
}
