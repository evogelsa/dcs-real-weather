.PHONY: release
release: update-licenses windows linux-amd64 linux-arm windows-bot linux-amd64-bot linux-arm-bot bundle-artifacts

.PHONY: update-licenses
update-licenses:
	go-licenses report ./... --template=oss-template.tmpl > oss-licenses.txt

.PHONY: generate
generate:
	go generate ./...

.PHONY: windows
windows: generate
	@echo "--------------------------------"
	@echo "Building for Windows (amd64)"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/windows"
	cd "cmd/realweather" && env GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/realweather.exe -trimpath
	cp config/config.toml bin/windows/config.toml
	cp oss-licenses.txt bin/windows/oss-licenses.txt
	zip -j windows.zip bin/windows/realweather.exe bin/windows/config.toml bin/windows/oss-licenses.txt
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	mv windows.zip bin/windows/realweather_$(VERSION).zip

.PHONY: linux-amd64
linux-amd64: generate
	@echo "--------------------------------"
	@echo "Building for Linux (amd64)"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/realweather -trimpath cmd/realweather/main.go
	cp config/config.toml bin/linux/config.toml
	cp oss-licenses.txt bin/linux/oss-licenses.txt
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/realweather_linux_amd64_$(VERSION).tar.gz -C bin/linux/ realweather config.toml oss-licenses.txt

.PHONY: linux-arm
linux-arm: generate
	@echo "--------------------------------"
	@echo "Building for Linux (arm)"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/realweather -trimpath cmd/realweather/main.go
	cp config/config.toml bin/linux/config.toml
	cp oss-licenses.txt bin/linux/oss-licenses.txt
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/realweather_linux_arm_$(VERSION).tar.gz -C bin/linux/ realweather config.toml oss-licenses.txt

.PHONY: windows-bot
windows-bot: generate
	@echo "--------------------------------"
	@echo "Building bot for Windows (amd64)"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/windows"
	cd "cmd/bot" && env GOOS=windows GOARCH=amd64 go build -o ../../bin/windows/rwbot.exe -trimpath
	cp cmd/bot/config/config.json bin/windows/botconfig.json
	cp oss-licenses.txt bin/windows/oss-licenses.txt
	zip -j windows.zip bin/windows/rwbot.exe bin/windows/botconfig.json bin/windows/oss-licenses.txt
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	mv windows.zip bin/windows/rwbot_$(VERSION).zip

.PHONY: linux-amd64-bot
linux-amd64-bot: generate
	@echo "--------------------------------"
	@echo "Building bot for Linux (amd64)"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=amd64 go build -o bin/linux/rwbot -trimpath cmd/bot/main.go
	cp cmd/bot/config/config.json bin/linux/botconfig.json
	cp oss-licenses.txt bin/linux/oss-licenses.txt
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/rwbot_linux_amd64_$(VERSION).tar.gz -C bin/linux/ rwbot botconfig.json oss-licenses.txt

.PHONY: linux-arm-bot
linux-arm-bot: generate
	@echo "--------------------------------"
	@echo "Building bot for Linux (arm)"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/linux"
	-rm resource.syso
	env GOOS=linux GOARCH=arm go build -o bin/linux/rwbot -trimpath cmd/bot/main.go
	cp cmd/bot/config/config.json bin/linux/botconfig.json
	cp oss-licenses.txt bin/linux/oss-licenses.txt
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	tar czf bin/linux/rwbot_linux_arm_$(VERSION).tar.gz -C bin/linux/ rwbot botconfig.json oss-licenses.txt

.PHONY: bundle-artifacts
bundle-artifacts:
	@echo "--------------------------------"
	@echo "Bundling all build artifacts"
	@echo "--------------------------------"
	-@mkdir "bin"
	-@mkdir "bin/bundle"
	$(eval VERSION := $(shell cat versioninfo/version.txt))
	-rm bin/bundle/*
	-cp bin/windows/realweather_$(VERSION).zip bin/bundle
	-cp bin/windows/rwbot_$(VERSION).zip bin/bundle
	-cp bin/linux/realweather_linux_amd64_$(VERSION).tar.gz bin/bundle
	-cp bin/linux/rwbot_linux_amd64_$(VERSION).tar.gz bin/bundle
	-cp bin/linux/realweather_linux_arm_$(VERSION).tar.gz bin/bundle
	-cp bin/linux/rwbot_linux_arm_$(VERSION).tar.gz bin/bundle
