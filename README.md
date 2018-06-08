# MQTT Stresser

Load testing tool to stress MQTT message broker

## Build

```
$ mkdir -p ${GOPATH}/src/github.com/inovex/
$ git clone https://github.com/inovex/mqtt-stresser.git ${GOPATH}/src/github.com/inovex/mqtt-stresser/
$ cd ${GOPATH}/src/github.com/inovex/mqtt-stresser/
$ make
```

This will build the mqtt stresser for all target platforms and write them to the ``build/`` directory.

Binaries are provided on Github, see https://github.com/inovex/mqtt-stresser.

If you want to build the Docker container version of this, go to repository directory and simply type ``docker build .``

## Install

Place the binary somewhere in a ``PATH`` directory and make it executable (``chmod +x mqtt-stresser``).

If you are using the container version, just type ``docker run flaviostutz/mqtt-stresser [options]`` for running mqtt-stresser.

## Configure

See ``mqtt-stresser -h`` for a list of available arguments.

## Run

Simple hello-world test using the public ``broker.mqttdashboard.com`` broker: (please don't DDoS them :))

```
$ mqtt-stresser -broker tcp://broker.mqttdashboard.com:1883 -num-clients 100 -num-messages 10 -rampup-delay 1s -rampup-size 10 -global-timeout 180s -timeout 20s
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
Messages / Client:  1000

# Results
Published Messages: 1000 (100%)
Received Messages:  1000 (100%)
Completed:          100 (100%)
Errors:             0 (0%)

# Publishing Throughput
Fastest: 31746 msg/sec
Slowest: 10106 msg/sec
Median: 23635 msg/sec

  < 31746 msg/sec  4%
  < 23090 msg/sec  22%
  < 18762 msg/sec  3%
  < 16598 msg/sec  1%
  < 14434 msg/sec  1%
  < 12270 msg/sec  1%
  < 29582 msg/sec  5%
  < 27418 msg/sec  21%
  < 25254 msg/sec  25%
  < 20926 msg/sec  17%

# Receiving Througput
Fastest: 491 msg/sec
Slowest: 33 msg/sec
Median: 293 msg/sec

  < 537 msg/sec  1%
  < 491 msg/sec  6%
  < 400 msg/sec  10%
  < 354 msg/sec  20%
  < 308 msg/sec  26%
  < 446 msg/sec  6%
  < 262 msg/sec  20%
  < 217 msg/sec  8%
  < 171 msg/sec  2%
  < 79 msg/sec  1%
```

If using container, 
```
$ docker run flaviostutz/mqtt-stresser -broker tcp://broker.mqttdashboard.com:1883 -num-clients 100 -num-messages 10 -rampup-delay 1s -rampup-size 10 -global-timeout 180s -timeout 20s
```