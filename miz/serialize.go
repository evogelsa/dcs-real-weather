package miz

import (
	"bytes"
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/evogelsa/DCS-real-weather/v2/logger"
)

func serializeTable(tbl *lua.LTable, indentLevel uint) string {
	var buf bytes.Buffer
	const indent = "\t"

	startString := "{\n"
	buf.WriteString(startString)

	tbl.ForEach(func(key lua.LValue, value lua.LValue) {
		// indent
		buf.WriteString(strings.Repeat(indent, int(indentLevel+1)))

		// serialize key
		switch key.Type() {
		case lua.LTString:
			buf.WriteString(fmt.Sprintf("[%q] = ", key.String()))
		case lua.LTNumber:
			buf.WriteString(fmt.Sprintf("[%v] = ", lua.LVAsNumber(key)))
		default:
			logger.Errorf("error serializing mission: unsupported key %v with type %s", key, key.Type().String())
		}

		// serialize value
		switch value.Type() {
		case lua.LTString:
			s := strings.ReplaceAll(value.String(), `\`, `\\`)
			s = strings.ReplaceAll(s, "\n", "\\\n")
			s = strings.ReplaceAll(s, "\"", "\\\"")
			s = fmt.Sprintf("\"%s\"", s)
			buf.WriteString(s)
		case lua.LTNumber:
			buf.WriteString(fmt.Sprintf("%v", lua.LVAsNumber(value)))
		case lua.LTBool:
			buf.WriteString(fmt.Sprintf("%t", lua.LVAsBool(value)))
		case lua.LTTable:
			// recursively serialize any tables
			buf.WriteString(serializeTable(value.(*lua.LTable), indentLevel+1))
		default:
			logger.Errorf("error serializing mission: unsupported value %v with type %s", value, value.Type().String())
		}

		buf.WriteString(",\n")
	})

	// if wrote more than just opening bracket, remove the last comma and \n
	// otherwise remove just the newline
	if buf.Len() > len(startString) {
		buf.Truncate(buf.Len() - 2)
		buf.WriteString("\n" + strings.Repeat(indent, int(indentLevel)) + "}")
	} else {
		buf.Truncate(buf.Len() - 1)
		buf.WriteString(" }")
	}

	return buf.String()
}
