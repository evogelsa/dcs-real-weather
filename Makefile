windows:
	mkdir -p bin/windows
	env GOOS=windows GOARCH=amd64 go build -o bin/windows/realweather.exe
