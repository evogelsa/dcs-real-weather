windows:
	mkdir -p bin/windows
	env GOOS=windows GOARCH=amd64 go build -o bin/windows/realweather.exe
	cp examples/config-template.json bin/windows/config.json
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.json
	mv windows.zip bin/windows/.
