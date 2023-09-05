windows:
	mkdir -p bin/windows
	go generate
	env GOOS=windows GOARCH=amd64 go build -o bin/windows/realweather.exe
	cp examples/config.json bin/windows/config.json
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.json
	mv windows.zip bin/windows/.

linux:
	mkdir -p bin/linux
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/realweather
	cp examples/config.json bin/linux/config.json
