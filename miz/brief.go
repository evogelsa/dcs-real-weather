package miz

import (
	"fmt"
	"os"
	"regexp"

	lua "github.com/yuin/gopher-lua"

	"github.com/evogelsa/DCS-real-weather/config"
	"github.com/evogelsa/DCS-real-weather/logger"
)

// UpdateBrief updates the unpacked mission brief with the generated METAR
func UpdateBrief(metar string) error {
	key := config.Get().RealWeather.Mission.Brief.InsertKey
	metarRE := regexp.MustCompile(key + "\n(?P<metar>.*)\n")

	logger.Infoln("loading mission brief into Lua VM...")

	// load brief into lua vm
	if err := l.DoFile("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("error loading mission dictionary: %v", err)
	}

	logger.Infoln("loaded mission brief into Lua VM")
	logger.Infoln("parsing mission brief for RW METAR insertion location...")

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
				logger.Infoln("appending METAR to brief")
				newBrief = brief.String() + "\n\n==Real Weather METAR==\n" + metar + "\n"
			}
		} else {
			logger.Errorln("unable to parse existing brief")
			logger.Warnln("writing new brief")
			newBrief = metar
		}
	} else {
		logger.Errorln("unable to parse existing brief")
		logger.Warnln("writing new brief")
		newBrief = metar
	}

	logger.Infoln("adding METAR to mission brief...")

	// write new brief
	if err := l.DoString(
		`dictionary.DictKey_descriptionText_1 = ` + fmt.Sprintf("%q", newBrief),
	); err != nil {
		return fmt.Errorf("error updating mission brief: %v", err)
	}

	// update brief by removing old and dumping lua state as new file

	if err := os.Remove("mission_unpacked/l10n/DEFAULT/dictionary"); err != nil {
		return fmt.Errorf("error removing mission dictionary: %v", err)
	}

	lv = l.GetGlobal("dictionary")
	if tbl, ok := lv.(*lua.LTable); ok {
		s := serializeTable(tbl, 0)
		s = "dictionary = " + s
		os.WriteFile("mission_unpacked/l10n/DEFAULT/dictionary", []byte(s), 0666)
	} else {
		return fmt.Errorf("error dumping serialized state")
	}

	logger.Infoln("added METAR to mission brief")

	return nil
}
