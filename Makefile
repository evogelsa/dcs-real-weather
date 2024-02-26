windows:
	mkdir -p bin/windows
	go generate ./...
	env GOOS=windows GOARCH=amd64 go build -o bin/windows/realweather.exe -trimpath
	cp examples/config.json bin/windows/config.json
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.json
	mv windows.zip bin/windows/.

linux-amd64:
	mkdir -p bin/linux
	go generate ./...
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/realweather -trimpath
	cp examples/config.json bin/linux/config.json
	tar czf bin/linux/realweather_linux_amd64.tar.gz bin/linux/realweather bin/linux/config.json

linux-arm:
	mkdir -p bin/linux
	go generate ./...
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/realweather -trimpath
	cp examples/config.json bin/linux/config.json
	tar czf bin/linux/realweather_linux_arm.tar.gz bin/linux/realweather bin/linux/config.json
