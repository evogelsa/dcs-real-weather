package miz

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/evogelsa/DCS-real-weather/config"

	lua "github.com/yuin/gopher-lua"
)

// UpdateBrief updates the unpacked mission brief with the generated METAR
func UpdateBrief(metar string) error {
	key := config.Get().RealWeather.Mission.Brief.InsertKey
	metarRE := regexp.MustCompile(key + "\n(?P<metar>.*)\n")

	log.Println("Loading mission brief into Lua VM...")

	// load brief into lua vm
	if err := l.DoFile("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("Error loading mission dictionary: %v", err)
	}

	log.Println("Loaded mission brief into Lua VM")
	log.Println("Parsing mission brief for RW METAR insertion location...")

	// parse brief dictionary for existing brief text
	lv := l.GetGlobal("dictionary")
	var newBrief string
	if dict, ok := lv.(*lua.LTable); ok {
		if brief, ok := dict.RawGetString("DictKey_descriptionText_1").(lua.LString); ok {
			// replace METAR after marker
			if key != "" && metarRE.MatchString(brief.String()) {
				newBrief = metarRE.ReplaceAllString(
					brief.String(),
					"==Real Weather METAR==\n"+metar+"\n",
				)
			} else {
				log.Println("METAR will be appended to brief")
				newBrief = brief.String() + "\n\n==Real Weather METAR==\n" + metar + "\n"
			}
		} else {
			log.Println("Unable to parse existing brief, new brief will be written")
			newBrief = metar
		}
	} else {
		log.Println("Unable to parse existing brief, new brief will be written")
		newBrief = metar
	}

	log.Println("Adding METAR to mission brief...")

	// write new brief
	if err := l.DoString(
		`dictionary.DictKey_descriptionText_1 = ` + fmt.Sprintf("%q", newBrief),
	); err != nil {
		return fmt.Errorf("Error updating mission brief: %v", err)
	}

	// update brief by removing old and dumping lua state as new file

	if err := os.Remove("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("Error removing mission dictionary: %v", err)
	}

	lv = l.GetGlobal("dictionary")
	if tbl, ok := lv.(*lua.LTable); ok {
		s := serializeTable(tbl, 0)
		s = "dictionary = " + s
		os.WriteFile("mission_unpacked/l10n/DEFAULT/dictionary", []byte(s), 0666)
	} else {
		return fmt.Errorf("Error dumping serialized state")
	}

	log.Println("Added METAR to mission brief")

	return nil
}
