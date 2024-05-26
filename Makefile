$(shell go generate ./...)
VERSION := $(shell cat versioninfo/version.txt)

all: windows linux-amd64 linux-arm windows-bot linux-amd64-bot linux-arm-bot

windows:
	@echo "--------------------------------"
	@echo "Building for Windows (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/windows"
	cd "cmd/realweather" && env GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/realweather.exe -trimpath
	cp config/config.json bin/windows/config.json
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.json
	mv windows.zip bin/windows/realweather_$(VERSION).zip

linux-amd64:
	@echo "--------------------------------"
	@echo "Building for Linux (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/realweather -trimpath cmd/realweather/main.go
	cp config/config.json bin/linux/config.json
	tar czf bin/linux/realweather_linux_amd64_$(VERSION).tar.gz -C bin/linux/ realweather config.json

linux-arm:
	@echo "--------------------------------"
	@echo "Building for Linux (arm)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/realweather -trimpath cmd/realweather/main.go
	cp config/config.json bin/linux/config.json
	tar czf bin/linux/realweather_linux_arm_$(VERSION).tar.gz -C bin/linux/ realweather config.json

windows-bot:
	@echo "--------------------------------"
	@echo "Building bot for Windows (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/windows"
	cd "cmd/bot" && env GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/rwbot.exe -trimpath
	cp cmd/bot/config/config.json bin/windows/botconfig.json
	zip -j windows.zip bin/windows/rwbot.exe bin/windows/botconfig.json
	mv windows.zip bin/windows/rwbot_$(VERSION).zip

linux-amd64-bot:
	@echo "--------------------------------"
	@echo "Building bot for Linux (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/rwbot -trimpath cmd/bot/main.go
	cp cmd/bot/config/config.json bin/linux/botconfig.json
	tar czf bin/linux/rwbot_linux_amd64_$(VERSION).tar.gz -C bin/linux/ rwbot botconfig.json

linux-arm-bot:
	@echo "--------------------------------"
	@echo "Building bot for Linux (arm)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/rwbot -trimpath cmd/bot/main.go
	cp cmd/bot/config/config.json bin/linux/botconfig.json
	tar czf bin/linux/rwbot_linux_arm_$(VERSION).tar.gz -C bin/linux/ rwbot botconfig.json
