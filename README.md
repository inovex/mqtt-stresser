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

This will build the mqtt stresser for all target platforms and write them to the ``build/`` directory. Binaries are provided on Github, see https://github.com/inovex/mqtt-stresser.


If you want to build your own Docker image, simply type ``make container`` or see ``Makefile`` for further ``docker build`` details.

## Install

Place the binary somewhere in a ``$PATH`` directory and make it executable (``chmod +x mqtt-stresser``).

## Configure

See ``mqtt-stresser -h`` for a list of available arguments.

## Run

Simple hello-world test using the public ``broker.mqttdashboard.com`` broker: (please don't DDoS them :))

```
$ mqtt-stresser -broker tcp://broker.mqttdashboard.com:1883 -num-clients 10 -num-messages 150 -rampup-delay 1s -rampup-size 10 -global-timeout 180s -timeout 20s

10 worker started
..........
# Configuration
Concurrent Clients: 10
Messages / Client:  1500

# Results
Published Messages: 1500 (100%)
Received Messages:  1500 (100%)
Completed:          10 (100%)
Errors:             0 (0%)

# Publishing Throughput
Fastest: 32431 msg/sec
Slowest: 16044 msg/sec
Median: 25483 msg/sec

  < 17683 msg/sec  20%
  < 24238 msg/sec  40%
  < 25876 msg/sec  60%
  < 29154 msg/sec  70%
  < 30792 msg/sec  90%
  < 32431 msg/sec  100%

# Receiving Througput
Fastest: 1469 msg/sec
Slowest: 214 msg/sec
Median: 552 msg/sec

  < 340 msg/sec  30%
  < 591 msg/sec  50%
  < 716 msg/sec  60%
  < 841 msg/sec  80%
  < 1469 msg/sec  90%
  < 1594 msg/sec  100%
```

Or using using our Docker image:

```
$ docker run --rm inovex/mqtt-stresser -broker tcp://broker.mqttdashboard.com:1883 -num-clients 10 -num-messages 150 -rampup-delay 1s -rampup-size 10 -global-timeout 180s -timeout 20s
```

## Develop

### Update dependencies
To update the used dependencies to the latest path version run:

```bash
make vendor-update vendor
```

If you want to bump the major version or set a specific version make your change in the `go.mod` file. Than run

```bash
make vendor
```

