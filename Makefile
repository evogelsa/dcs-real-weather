.PHONY: release
release: package-windows package-linux-amd64 package-linux-arm package-windows-bot package-linux-amd64-bot package-linux-arm-bot

.PHONY: generate
generate:
	go generate ./...

.PHONY: windows
windows: generate
	@echo "--------------------------------"
	@echo "Building for Windows (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/windows"
	cd "cmd/realweather" && env GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/realweather.exe -trimpath

.PHONY: package-windows
package-windows: windows
	sleep 5
	cp config/config.json bin/windows/config.json
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.json
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	mv windows.zip bin/windows/realweather_$(VERSION).zip

.PHONY: linux-amd64
linux-amd64: generate
	@echo "--------------------------------"
	@echo "Building for Linux (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/realweather -trimpath cmd/realweather/main.go

.PHONY: package-linux-amd64
package-linux-amd64: linux-amd64
	sleep 5
	cp config/config.json bin/linux/config.json
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/realweather_linux_amd64_$(VERSION).tar.gz -C bin/linux/ realweather config.json

.PHONY: linux-arm
linux-arm: generate
	@echo "--------------------------------"
	@echo "Building for Linux (arm)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/realweather -trimpath cmd/realweather/main.go

.PHONY: package-linux-arm
package-linux-arm: linux-arm
	sleep 5
	cp config/config.json bin/linux/config.json
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/realweather_linux_arm_$(VERSION).tar.gz -C bin/linux/ realweather config.json

.PHONY: windows-bot
windows-bot: generate
	@echo "--------------------------------"
	@echo "Building bot for Windows (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/windows"
	cd "cmd/bot" && env GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/rwbot.exe -trimpath

.PHONY: package-windows-bot
package-windows-bot: windows-bot
	sleep 5
	cp cmd/bot/config/config.json bin/windows/botconfig.json
	zip -j windows.zip bin/windows/rwbot.exe bin/windows/botconfig.json
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	mv windows.zip bin/windows/rwbot_$(VERSION).zip

.PHONY: linux-amd64-bot
linux-amd64-bot: generate
	@echo "--------------------------------"
	@echo "Building bot for Linux (amd64)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/rwbot -trimpath cmd/bot/main.go

.PHONY: package-linux-amd64-bot
package-linux-amd64-bot: generate
	sleep 5
	cp cmd/bot/config/config.json bin/linux/botconfig.json
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/rwbot_linux_amd64_$(VERSION).tar.gz -C bin/linux/ rwbot botconfig.json

.PHONY: linux-arm-bot
linux-arm-bot: generate
	@echo "--------------------------------"
	@echo "Building bot for Linux (arm)"
	@echo "--------------------------------"
	-mkdir "bin"
	-mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/rwbot -trimpath cmd/bot/main.go

.PHONY: package-linux-arm-bot
package-linux-arm-bot: linux-arm-bot
	sleep 5
	cp cmd/bot/config/config.json bin/linux/botconfig.json
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/rwbot_linux_arm_$(VERSION).tar.gz -C bin/linux/ rwbot botconfig.json
