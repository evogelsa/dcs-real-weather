package miz

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// TestSerializeTable calls serializeTable with a lua snippet and tests that the
// serialized form is the same as the input form
func TestSerializeTable(t *testing.T) {
	const input = `mission = {
	["params"] = {
		["command"] = "local gr = ...\
gr:destroy()\
trigger.action.outTextForCoalition(2,'Enemy cargo plane has landed',15)"
	}
}`
	var output string

	var l *lua.LState
	l = lua.NewState()

	l.DoString(input)
	lv := l.GetGlobal("mission")
	if tbl, ok := lv.(*lua.LTable); ok {
		s := serializeTable(tbl, 0)
		s = "mission = " + s
		output = s
	} else {
		t.Fatalf("bad test case")
	}

	if input != output {
		t.Fatalf("got\n%#q\n\nexpected\n%#q", output, input)
	}
}
