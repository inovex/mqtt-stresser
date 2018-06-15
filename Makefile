appname := mqtt-stresser
namespace := inovex
sources := $(wildcard *.go)

build = GOOS=$(1) GOARCH=$(2) go build -o build/$(appname)$(3)
static-build = CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/$(appname) .
tar = cd build && tar -cvzf mqtt-stresser-$(1)-$(2).tar.gz $(appname)$(3) && rm $(appname)$(3)
zip = cd build && zip mqtt-stresser-$(1)-$(2).zip $(appname)$(3) && rm $(appname)$(3)

.PHONY: all windows darwin linux clean container push-container

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

linux-static: $(sources)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/$(appname).static .

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


##### DOCKER #####
container:
	docker build \
		--build-arg BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")\
		--build-arg VCS_REF=$(shell git rev-parse --short HEAD) \
	 	--build-arg VERSION=$(shell git describe --all | sed  -e  's%tags/%%g'  -e 's%/%.%g' ) \
	 	-t $(namespace)/$(appname) .

push-container:
	docker tag $(namespace)/$(appname) $(namespace)/$(appname):$(VERSION)
	docker push $(namespace)/$(appname):$(VERSION)

