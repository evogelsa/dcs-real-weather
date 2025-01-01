package miz

import (
	lua "github.com/yuin/gopher-lua"
)

var l *lua.LState

func init() {
	l = lua.NewState(lua.Options{
		RegistrySize:     1024,
		RegistryMaxSize:  1024 * 1024,
		RegistryGrowStep: 1024,
	})
}
