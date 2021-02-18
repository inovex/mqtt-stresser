package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

var (
	resultChan   = make(chan Result)
	stopWaitLoop = false
	randomSource = rand.New(rand.NewSource(time.Now().UnixNano()))

	subscriberClientIdTemplate = "mqtt-stresser-sub-%s-worker%d-%d"
	publisherClientIdTemplate  = "mqtt-stresser-pub-%s-worker%d-%d"
	topicNameTemplate          = "internal/mqtt-stresser/%s/worker%d-%d"

	errorLogger   = log.New(os.Stderr, "ERROR: ", log.Lmicroseconds|log.Ltime|log.Lshortfile)
	verboseLogger = log.New(os.Stderr, "DEBUG: ", log.Lmicroseconds|log.Ltime|log.Lshortfile)

	argNumClients           = flag.Int("num-clients", 10, "Number of concurrent clients")
	argNumMessages          = flag.Int("num-messages", 10, "Number of messages shipped by client")
	argConstantPayload      = flag.String("constant-payload", "", "Use this constant payload in every MQTT message. If not set, an nearly constant autogenerated payload is used.")
	argTimeout              = flag.String("timeout", "5s", "Timeout for pub/sub actions")
	argGlobalTimeout        = flag.String("global-timeout", "60s", "Timeout spanning all operations")
	argRampUpSize           = flag.Int("rampup-size", 100, "Size of rampup batch. Default rampup batch size is 100.")
	argRampUpDelay          = flag.String("rampup-delay", "500ms", "Time between batch rampups")
	argBrokerUrl            = flag.String("broker", "", "Broker URL")
	argUsername             = flag.String("username", "", "Username")
	argPassword             = flag.String("password", "", "Password")
	argLogLevel             = flag.Int("log-level", 0, "Log level (0=nothing, 1=errors, 2=debug, 3=error+debug)")
	argProfileCpu           = flag.String("profile-cpu", "", "write cpu profile `file`")
	argProfileMem           = flag.String("profile-mem", "", "write memory profile to `file`")
	argHideProgress         = flag.Bool("no-progress", false, "Hide progress indicator")
	argHelp                 = flag.Bool("help", false, "Show help")
	argRetain               = flag.Bool("retain", false, "if set, the retained flag of the published mqtt messages is set")
	argPublisherQoS         = flag.Int("publisher-qos", 0, "QoS level of published messages")
	argSubscriberQoS        = flag.Int("subscriber-qos", 0, " QoS level for the subscriber")
	argSkipTLSVerification  = flag.Bool("skip-tls-verification", false, "skip the tls verfication of the MQTT Connection")
	argCafile               = flag.String("cafile", "", "path to a file containing trusted CA certificates to enable encrypted certificate based communication.")
	argKey                  = flag.String("key", "", "client private key for authentication, if required by server.")
	argCert                 = flag.String("cert", "", "client certificate for authentication, if required by server.")
	argPauseBetweenMessages = flag.String("pause-between-messages", "0s", "Adds a pause between sending messages to simulate sensors sending messages infrequently")
	argTopicBasePath		= flag.String("topic-base-path", "", "topic base path, if empty the default is internal/mqtt-stresser")
)

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

type TimeoutError interface {
	Timeout() bool
	Error() string
}

func parseQosLevels(qos int, role string) (byte, error) {
	if qos < 0 || qos > 2 {
		return 0, fmt.Errorf("%q is an invalid QoS level for %s. Valid levels are 0, 1 and 2", qos, role)
	}
	return byte(qos), nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// An error is returned of the given TLS configuration is invalid.
func validateTLSFiles(argCafile, argKey, argCert string) error {
	if len(argCafile) > 0 {
		if !fileExists(argCafile) {
			return fmt.Errorf("CA file %q does not exist", argCafile)
		}
	}
	if len(argKey) > 0 {
		if !fileExists(argKey) {
			return fmt.Errorf("key file %q does not exist", argKey)
		}
	}
	if len(argCert) > 0 {
		if !fileExists(argCert) {
			return fmt.Errorf("cert file %q does not exist", argCert)
		}
	}

	if len(argKey) > 0 && len(argCert) < 1 {
		return fmt.Errorf("A key file is specified but no certificate file")
	}

	if len(argKey) < 1 && len(argCert) > 0 {
		return fmt.Errorf("A cert file is specified but no key file")
	}
	return nil
}

// loadTLSFile loads the given file. If the filename is empty neither data nor an error is returned.
func loadTLSFile(fileName string) ([]byte, error) {
	if len(fileName) > 0 {
		data, err := ioutil.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS file: %q: %w", fileName, err)
		}
		return data, nil
	}
	return nil, nil
}

func main() {
	flag.Parse()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":2112", nil)
		if err != nil {
			fmt.Printf("Failed to init prometheus metrics port: %v\n", err)
		}
	}()

	if flag.NFlag() < 1 || *argHelp {
		flag.Usage()
		if *argHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if *argProfileCpu != "" {
		f, err := os.Create(*argProfileCpu)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create CPU profile: %s\n", err)
			os.Exit(1)
		}

		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start CPU profile: %s\n", err)
			os.Exit(1)
		}
	}

	num := *argNumMessages
	username := *argUsername
	password := *argPassword

	actionTimeout, err := time.ParseDuration(*argTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse '--timeout': %q is not a valid duration string. See https://golang.org/pkg/time/#ParseDuration for valid duration strings\n", *argTimeout)
		os.Exit(1)
	}

	verboseLogger.SetOutput(ioutil.Discard)
	errorLogger.SetOutput(ioutil.Discard)

	if *argLogLevel == 1 || *argLogLevel == 3 {
		errorLogger.SetOutput(os.Stderr)
	}

	if *argLogLevel == 2 || *argLogLevel == 3 {
		verboseLogger.SetOutput(os.Stderr)
	}

	if *argBrokerUrl == "" {
		fmt.Fprintln(os.Stderr, "'--broker' is empty. Abort.")
		os.Exit(1)
	}

	if len(*argTopicBasePath) > 0 {
		topicNameTemplate = strings.Replace(topicNameTemplate, "internal/mqtt-stresser", *argTopicBasePath, 1)
	}

	payloadGenerator := defaultPayloadGen()
	if len(*argConstantPayload) > 0 {
		if strings.HasPrefix(*argConstantPayload, "@") {
			verboseLogger.Printf("Set constant payload from file %s\n", *argConstantPayload)
			payloadGenerator = filePayloadGenerator(*argConstantPayload)
		}else {
			verboseLogger.Printf("Set constant payload to %s\n", *argConstantPayload)
			payloadGenerator = constantPayloadGenerator(*argConstantPayload)
		}
	}

	var publisherQoS, subscriberQoS byte

	if lvl, err := parseQosLevels(*argPublisherQoS, "publisher"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		publisherQoS = lvl
	}

	if lvl, err := parseQosLevels(*argSubscriberQoS, "subscriber"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		subscriberQoS = lvl
	}

	var ca, cert, key []byte
	if err := validateTLSFiles(*argCafile, *argKey, *argCert); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ca, err = loadTLSFile(*argCafile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cert, err = loadTLSFile(*argCert)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	key, err = loadTLSFile(*argKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	globalTimeout, err := time.ParseDuration(*argGlobalTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed parse '--global-timeout': %q is not a valid duration string. See https://golang.org/pkg/time/#ParseDuration for valid duration strings\n", *argGlobalTimeout)
		os.Exit(1)
	}
	testCtx, cancelFunc := context.WithTimeout(context.Background(), globalTimeout)

	pauseBetweenMessages, err := time.ParseDuration(*argPauseBetweenMessages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed parse '--pause-between-messages': %q is not a valid duration string. See https://golang.org/pkg/time/#ParseDuration for valid duration strings\n", *argPauseBetweenMessages)
		os.Exit(1)
	}

	stopStartLoop := false
	for cid := 0; cid < *argNumClients && !stopStartLoop; cid++ {

		if cid%rampUpSize == 0 && cid > 0 {
			fmt.Printf("%d worker started - waiting %s\n", cid, rampUpDelay)
			select {
			case <-time.NewTimer(rampUpDelay).C:
			case s := <-signalChan:
				fmt.Printf("Got signal %s. Cancel test.\n", s.String())
				cancelFunc()
				stopStartLoop = true
			}
		}

		go (&Worker{
			WorkerId:             cid,
			BrokerUrl:            *argBrokerUrl,
			Username:             username,
			Password:             password,
			SkipTLSVerification:  *argSkipTLSVerification,
			NumberOfMessages:     num,
			PayloadGenerator:     payloadGenerator,
			Timeout:              actionTimeout,
			Retained:             *argRetain,
			PublisherQoS:         publisherQoS,
			SubscriberQoS:        subscriberQoS,
			CA:                   ca,
			Cert:                 cert,
			Key:                  key,
			PauseBetweenMessages: pauseBetweenMessages,
		}).Run(testCtx)
	}
	fmt.Printf("%d worker started\n", *argNumClients)

	finEvents := 0

	results := make([]Result, *argNumClients)

	for finEvents < *argNumClients && !stopWaitLoop {
		select {
		case msg := <-resultChan:
			results[msg.WorkerId] = msg

			if msg.Event == CompletedEvent || msg.Error {
				finEvents++
				verboseLogger.Printf("%d/%d events received\n", finEvents, *argNumClients)
			}

			if msg.Error {
				errorLogger.Println(msg)
			}

			if !*argHideProgress {
				if msg.Event == ProgressReportEvent {
					fmt.Print(".")
				}

				if msg.Error {
					fmt.Print("E")
				}
			}

		case <-testCtx.Done():
			switch testCtx.Err().(type) {
			case TimeoutError:
				fmt.Println("Test timeout. Wait 5s to allow disconnection of clients.")
			default:
				fmt.Println("Test canceled. Wait 5s to allow disconnection of clients.")
			}
			time.Sleep(5 * time.Second)
			stopWaitLoop = true
		case s := <-signalChan:
			fmt.Printf("Got signal %s. Cancel test.\n", s.String())
			cancelFunc()
			stopWaitLoop = true
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
			fmt.Printf("Failed to create memory profile: %s\n", err)
		}

		runtime.GC() // get up-to-date statistics

		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Printf("Failed to write memory profile: %s\n", err)
		}
		f.Close()
	}

	pprof.StopCPUProfile()

	os.Exit(exitCode)
}
