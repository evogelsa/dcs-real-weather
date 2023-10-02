package versioninfo

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
)

//go:generate sh -c "printf %s $(git describe --tags) > git.txt"
//go:embed git.txt
var gitInfo string

const versionInfo = `{
	"FixedFileInfo": {
	  "FileVersion": {
		"Major": %s,
		"Minor": %s,
		"Patch": %s,
		"Build": %s
	  },
	  "ProductVersion": {
		"Major": %s,
		"Minor": %s,
		"Patch": %s,
		"Build": %s
	  }
	},
	"StringFileInfo": {
		"Comments": "",
		"CompanyName": "github.com/evogelsa",
		"FileDescription": "DCS Real Weather Updater",
		"FileVersion": "%s.%s.%s",
		"InternalName": "DCS Real Weather",
		"LegalCopyright": "Copyright 2020 evogelsa",
		"OriginalFilename": "realweather.exe",
		"ProductName": "DCS Real Weather",
		"ProductVersion": "%s.%s.%s.%s"
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

var re = regexp.MustCompile(`v(?P<Major>\d+)\.(?P<Minor>\d+)\.(?P<Patch>\d+)-?(?P<CommitNum>\d*)(?:-g)?(?P<Commit>\w*)`)

var (
	Major  string
	Minor  string
	Patch  string
	Build  string
	Commit string
)

func init() {
	match := re.FindStringSubmatch(gitInfo)
	v := make(map[string]string)

	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			v[name] = match[i]
		}
	}

	Major = v["Major"]
	Minor = v["Minor"]
	Patch = v["Patch"]
	Build = v["CommitNum"]
	Commit = v["Commit"]

	os.WriteFile(
		"versioninfo/versioninfo.json",
		[]byte(fmt.Sprintf(
			versionInfo,
			Major,
			Minor,
			Patch,
			Build,
			Major,
			Minor,
			Patch,
			Build,
			Major,
			Minor,
			Patch,
			Major,
			Minor,
			Patch,
			Build,
		)),
		os.ModePerm,
	)
}
