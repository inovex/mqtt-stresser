appname := mqtt-stresser
namespace := inovex
sources := vendor $(wildcard *.go)

build = GO111MODULE=on GOOS=$(1) GOARCH=$(2) go build -mod=vendor -o build/$(appname)-$(1)-$(2)$(3)
static-build = GO111MODULE=on CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) GOARM=$(4) go build -mod=vendor -a -installsuffix cgo -o build/$(appname)-$(1)-$(2)$(4)-static$(3) .
tar = cd build && tar -cvzf $(appname)-$(1)-$(2).tar.gz $(appname)-$(1)-$(2)$(3) && rm $(appname)-$(1)-$(2)$(3)
zip = cd build && zip $(appname)-$(1)-$(2).zip $(appname)-$(1)-$(2)$(3) && rm $(appname)-$(1)-$(2)$(3)

.PHONY: all windows darwin linux clean container push-container windows-static darwin-static linux-static windows-compressed linux-compressed darwin-compressed

all: windows darwin linux windows-static darwin-static linux-static-amd64 linux-static-arm5 linux-static-arm6 linux-static-arm7

all-static: windows-static darwin-static linux-static

release-files: windows-compressed linux-compressed darwin-compressed

clean:
	rm -rf build/

fmt:
	@gofmt -l -w $(sources)


##### LINUX #####
linux: build/$(appname)-linux-amd64

linux-compressed: build/$(appname)-linux-amd64.tar.gz

linux-static-amd64: build/$(appname)-linux-amd64-static
linux-static-arm5: build/$(appname)-linux-arm5-static
linux-static-arm6: build/$(appname)-linux-arm6-static
linux-static-arm7: build/$(appname)-linux-arm7-static

build/$(appname)-linux-amd64: $(sources)
	$(call build,linux,amd64,)

build/$(appname)-linux-amd64.tar.gz: build/$(appname)-linux-amd64
	$(call tar,linux,amd64)

build/$(appname)-linux-amd64-static: $(sources)
	$(call static-build,linux,amd64,)

build/$(appname)-linux-arm5-static: $(sources)
	$(call static-build,linux,arm,,5)

build/$(appname)-linux-arm6-static: $(sources)
	$(call static-build,linux,arm,,6)

build/$(appname)-linux-arm7-static: $(sources)
	$(call static-build,linux,arm,,7)


##### DARWIN (MAC) #####
darwin: build/$(appname)-darwin-amd64

darwin-compressed: build/$(appname)-darwin-amd64.tar.gz

darwin-static: build/$(appname)-darwin-amd64-static

build/$(appname)-darwin-amd64: $(sources)
	$(call build,darwin,amd64,)

build/$(appname)-darwin-amd64.tar.gz: build/$(appname)-darwin-amd64
	$(call tar,darwin,amd64)


build/$(appname)-darwin-amd64-static: $(sources)
	$(call static-build,darwin,amd64,)


##### WINDOWS #####

windows: build/$(appname)-windows-amd64.exe

windows-compressed: build/$(appname)-windows-amd64.zip

windows-static: build/$(appname)-windows-amd64-static

build/$(appname)-windows-amd64.exe: $(sources)
	$(call build,windows,amd64,.exe)

build/$(appname)-windows-amd64.zip: build/$(appname)-windows-amd64
	$(call zip,windows,amd64,.exe)



build/$(appname)-windows-amd64-static: $(sources)
	$(call static-build,windows,amd64,)


##### DOCKER #####
container: clean
	docker build \
		--build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")\
		--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
	 	--build-arg VERSION=$(shell git describe --all | sed -e 's%tags/%%g' -e 's%/%.%g' ) \
	 	-t $(namespace)/$(appname) .

	 	docker run -it --rm $(namespace)/$(appname) --help


push-container:
	docker tag $(namespace)/$(appname) $(namespace)/$(appname):$(VERSION)
	docker push $(namespace)/$(appname):$(VERSION)

##### Vendoring #####


go.mod:
	go mod init


vendor-update: go.mod
	GO111MODULE=on  go get -u=patch

vendor: go.mod
	GO111MODULE=on go mod vendor

