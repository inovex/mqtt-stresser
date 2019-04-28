# MQTT Stresser

Load testing tool to stress MQTT message broker

## Build

You need at least golang version 1.12 to build the binaries.
```
$ mkdir -p ${GOPATH}/src/github.com/inovex/
$ git clone https://github.com/inovex/mqtt-stresser.git ${GOPATH}/src/github.com/inovex/mqtt-stresser/
$ cd ${GOPATH}/src/github.com/inovex/mqtt-stresser/
$ make
```

This will build the mqtt stresser for all target platforms and write them to the ``build/`` directory.

Binaries are provided on Github, see https://github.com/inovex/mqtt-stresser.

If you want to build the Docker container version of this, go to repository directory and simply type ``docker build .``

### Update dependencies
To update the used dependencies to the latest path version run:
```bash
make vendor-update vendor
```
If you want to bump the major version or set a specific version make your change in the `go.mod` file. Than run 
```bash
make vendor
```

## Install

Place the binary somewhere in a ``PATH`` directory and make it executable (``chmod +x mqtt-stresser``).

If you are using the container version, just type ``docker run flaviostutz/mqtt-stresser [options]`` for running mqtt-stresser.

## Configure

See ``mqtt-stresser -h`` for a list of available arguments.

## Run

Simple hello-world test using the public ``broker.mqttdashboard.com`` broker: (please don't DDoS them :))

```
$ mqtt-stresser -broker tcp://broker.mqttdashboard.com:1883 -num-clients 100 -num-messages 150 -rampup-delay 1s -rampup-size 10 -global-timeout 180s -timeout 20s

10 worker started - waiting 1s
20 worker started - waiting 1s
30 worker started - waiting 1s
40 worker started - waiting 1s
50 worker started - waiting 1s
60 worker started - waiting 1s
70 worker started - waiting 1s
80 worker started - waiting 1s
90 worker started - waiting 1s
100 worker started
....................................................................................................
# Configuration
Concurrent Clients: 100
Messages / Client:  15000

# Results
Published Messages: 15000 (100%)
Received Messages:  15000 (100%)
Completed:          100 (100%)
Errors:             0 (0%)

# Publishing Throughput
Fastest: 79452 msg/sec
Slowest: 14991 msg/sec
Median: 42093 msg/sec

  < 21437 msg/sec  6%
  < 27883 msg/sec  21%
  < 34329 msg/sec  33%
  < 40776 msg/sec  48%
  < 47222 msg/sec  57%
  < 53668 msg/sec  65%
  < 60114 msg/sec  73%
  < 66560 msg/sec  85%
  < 73006 msg/sec  95%
  < 79452 msg/sec  99%
  < 85898 msg/sec  100%

# Receiving Througput
Fastest: 4102 msg/sec
Slowest: 65 msg/sec
Median: 1919 msg/sec

  < 469 msg/sec  33%
  < 1276 msg/sec  34%
  < 1680 msg/sec  38%
  < 2083 msg/sec  62%
  < 2487 msg/sec  85%
  < 2891 msg/sec  93%
  < 3295 msg/sec  98%
  < 4102 msg/sec  99%
  < 4506 msg/sec  100%
```

If using container, 
```
$ docker run inovex/mqtt-stresser -broker tcp://broker.mqttdashboard.com:1883 -num-clients 100 -num-messages 10 -rampup-delay 1s -rampup-size 10 -global-timeout 180s -timeout 20s
```
