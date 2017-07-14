package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"
)

var (
	resultChan         = make(chan Result)
	abortChan          = make(chan bool)
	stopWaitLoop       = false
	tearDownInProgress = false
	randomSource       = rand.New(rand.NewSource(time.Now().UnixNano()))

	subscriberClientIdTemplate = "mqtt-stresser-sub-%s-worker%d-%d"
	publisherClientIdTemplate  = "mqtt-stresser-pub-%s-worker%d-%d"
	topicNameTemplate          = "internal/mqtt-stresser/%s/worker%d-%d"

	opTimeout = 5 * time.Second

	errorLogger   = log.New(os.Stderr, "ERROR: ", log.Lmicroseconds|log.Ltime|log.Lshortfile)
	verboseLogger = log.New(os.Stderr, "DEBUG: ", log.Lmicroseconds|log.Ltime|log.Lshortfile)

	argNumClients    = flag.Int("num-clients", 10, "Number of concurrent clients")
	argNumMessages   = flag.Int("num-messages", 10, "Number of messages shipped by client")
	argTimeout       = flag.String("timeout", "5s", "Timeout for pub/sub loop")
	argGlobalTimeout = flag.String("global-timeout", "60s", "Timeout spanning all operations")
	argRampUpSize    = flag.Int("rampup-size", 100, "Size of rampup batch")
	argRampUpDelay   = flag.String("rampup-delay", "500ms", "Time between batch rampups")
	argTearDownDelay = flag.String("teardown-delay", "5s", "Graceperiod to complete remaining workers")
	argBrokerUrl     = flag.String("broker", "", "Broker URL")
	argUsername      = flag.String("username", "", "Username")
	argPassword      = flag.String("password", "", "Password")
	argLogLevel      = flag.Int("log-level", 0, "Log level (0=nothing, 1=errors, 2=debug, 3=error+debug)")
	argProfileCpu    = flag.String("profile-cpu", "", "write cpu profile `file`")
	argProfileMem    = flag.String("profile-mem", "", "write memory profile to `file`")
	argHideProgress  = flag.Bool("no-progress", false, "Hide progress indicator")
	argHelp          = flag.Bool("help", false, "Show help")
)

type Worker struct {
	WorkerId  int
	BrokerUrl string
	Username  string
	Password  string
	Nmessages int
	Timeout   time.Duration
}

type Result struct {
	WorkerId          int
	Event             string
	PublishTime       time.Duration
	ReceiveTime       time.Duration
	MessagesReceived  int
	MessagesPublished int
	Error             bool
	ErrorMessage      error
}

func main() {
	flag.Parse()

	if flag.NFlag() < 1 || *argHelp {
		flag.Usage()
		os.Exit(1)
	}

	if *argProfileCpu != "" {
		f, err := os.Create(*argProfileCpu)

		if err != nil {
			fmt.Printf("Could not create CPU profile: %s\n", err)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("Could not start CPU profile: %s\n", err)
		}
	}

	num := *argNumMessages
	brokerUrl := *argBrokerUrl
	username := *argUsername
	password := *argPassword
	testTimeout, _ := time.ParseDuration(*argTimeout)

	verboseLogger.SetOutput(ioutil.Discard)
	errorLogger.SetOutput(ioutil.Discard)

	if *argLogLevel == 1 || *argLogLevel == 3 {
		errorLogger.SetOutput(os.Stderr)
	}

	if *argLogLevel == 2 || *argLogLevel == 3 {
		verboseLogger.SetOutput(os.Stderr)
	}

	if brokerUrl == "" {
		os.Exit(1)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	rampUpDelay, _ := time.ParseDuration(*argRampUpDelay)
	rampUpSize := *argRampUpSize

	if rampUpSize < 0 {
		rampUpSize = 100
	}

	resultChan = make(chan Result, *argNumClients**argNumMessages)

	for cid := 0; cid < *argNumClients; cid++ {

		if cid%rampUpSize == 0 && cid > 0 {
			fmt.Printf("%d worker started - waiting %s\n", cid, rampUpDelay)
			time.Sleep(rampUpDelay)
		}

		go (&Worker{
			WorkerId:  cid,
			BrokerUrl: brokerUrl,
			Username:  username,
			Password:  password,
			Nmessages: num,
			Timeout:   testTimeout,
		}).Run()
	}
	fmt.Printf("%d worker started\n", *argNumClients)

	finEvents := 0

	timeout := make(chan bool, 1)
	globalTimeout, _ := time.ParseDuration(*argGlobalTimeout)
	results := make([]Result, *argNumClients)

	go func() {
		time.Sleep(globalTimeout)
		timeout <- true
	}()

	for finEvents < *argNumClients && !stopWaitLoop {
		select {
		case msg := <-resultChan:
			results[msg.WorkerId] = msg

			if msg.Event == "Completed" || msg.Error {
				finEvents++
				verboseLogger.Printf("%d/%d events received\n", finEvents, *argNumClients)
			}

			if msg.Error {
				errorLogger.Println(msg)
			}

			if *argHideProgress == false {
				if msg.Event == "Completed" {
					fmt.Print(".")
				}

				if msg.Error {
					fmt.Print("E")
				}
			}

		case <-timeout:
			fmt.Println()
			fmt.Printf("Aborted because global timeout (%s) was reached.\n", *argGlobalTimeout)

			go tearDownWorkers()
		case signal := <-signalChan:
			fmt.Println()
			fmt.Printf("Received %s. Aborting.\n", signal)

			go tearDownWorkers()
		}
	}

	summary, err := buildSummary(*argNumClients, num, results)
	exitCode := 0

	if err != nil {
		exitCode = 1
	} else {
		printSummary(summary)
	}

	if *argProfileMem != "" {
		f, err := os.Create(*argProfileMem)

		if err != nil {
			fmt.Printf("Could not create memory profile: %s\n", err)
		}

		runtime.GC() // get up-to-date statistics

		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Printf("Could not write memory profile: %s\n", err)
		}
		f.Close()
	}

	pprof.StopCPUProfile()

	os.Exit(exitCode)
}

func tearDownWorkers() {
	if !tearDownInProgress {
		tearDownInProgress = true

		close(abortChan)

		delay, _ := time.ParseDuration(*argTearDownDelay)
		fmt.Printf("Waiting %s for remaining workers\n", delay)
		time.Sleep(delay)

		stopWaitLoop = true
	}
}
