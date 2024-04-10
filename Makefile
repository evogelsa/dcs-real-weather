$(shell go generate ./...)
$(shell go run versioninfo/update/update.go)
VERSION := $(shell cat versioninfo/version.txt)

all: windows linux-amd64 linux-arm

windows:
	@echo "------------------------------"
	@echo "Building for Windows (amd64)"
	@echo "------------------------------"
	mkdir -p bin/windows
	env GOOS=windows GOARCH=amd64 go build -o bin/windows/realweather.exe -trimpath
	cp examples/config.json bin/windows/config.json
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.json
	mv windows.zip bin/windows/realweather_$(VERSION).zip

linux-amd64:
	@echo "------------------------------"
	@echo "Building for Linux (amd64)"
	@echo "------------------------------"
	mkdir -p bin/linux
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/realweather -trimpath
	cp examples/config.json bin/linux/config.json
	tar czf bin/linux/realweather_linux_amd64_$(VERSION).tar.gz -C bin/linux/ realweather config.json

linux-arm:
	@echo "------------------------------"
	@echo "Building for Linux (arm)"
	@echo "------------------------------"
	mkdir -p bin/linux
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/realweather -trimpath
	cp examples/config.json bin/linux/config.json
	tar czf bin/linux/realweather_linux_arm_$(VERSION).tar.gz -C bin/linux/ realweather config.json
