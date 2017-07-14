appname := mqtt-stresser

sources := $(wildcard *.go)

build = GOOS=$(1) GOARCH=$(2) go build -o build/$(appname)$(3)
tar = cd build && tar -cvzf mqtt-stresser-$(1)-$(2).tar.gz $(appname)$(3) && rm $(appname)$(3)
zip = cd build && zip mqtt-stresser-$(1)-$(2).zip $(appname)$(3) && rm $(appname)$(3)

.PHONY: all windows darwin linux clean

all: windows darwin linux

clean:
	rm -rf build/

fmt:
	@gofmt -l -w $(sources)

vendor-deps:
	@echo ">> Fetching dependencies"
	go get github.com/rancher/trash

vendor: vendor-deps
	rm -r vendor/
	${GOPATH}/bin/trash -u
	${GOPATH}/bin/trash

##### LINUX #####
linux: build/mqtt-stresser-linux-amd64.tar.gz

build/mqtt-stresser-linux-amd64.tar.gz: $(sources)
	$(call build,linux,amd64,)
	$(call tar,linux,amd64)

##### DARWIN (MAC) #####
darwin: build/mqtt-stresser-darwin-amd64.tar.gz

build/mqtt-stresser-darwin-amd64.tar.gz: $(sources)
	$(call build,darwin,amd64,)
	$(call tar,darwin,amd64)

##### WINDOWS #####
windows: build/mqtt-stresser-windows-amd64.zip

build/mqtt-stresser-windows-amd64.zip: $(sources)
	$(call build,windows,amd64,.exe)
	$(call zip,windows,amd64,.exe)

