package main

import (
	"fmt"
	"os"

	"github.com/evogelsa/DCS-real-weather/versioninfo"
)

func main() {
	var ver string

	ver += fmt.Sprintf("v%d.%d.%d", versioninfo.Major, versioninfo.Minor, versioninfo.Patch)
	if versioninfo.Pre != "" {
		ver += fmt.Sprintf("-%s-%d", versioninfo.Pre, versioninfo.CommitNum)
	}
	if versioninfo.Commit != "" {
		ver += "+" + versioninfo.Commit
	}

	os.WriteFile("versioninfo/version.txt", []byte(ver), 0666)
}
