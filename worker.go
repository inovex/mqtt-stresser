package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

type PayloadGenerator func(i int) string

func defaultPayloadGen() PayloadGenerator {
	return func(i int) string {
		return fmt.Sprintf("this is msg #%d!", i)
	}
}

func constantPayloadGenerator(payload string) PayloadGenerator {
	return func(i int) string {
		return payload
	}
}

func filePayloadGenerator(filepath string) PayloadGenerator {
	inputPath := strings.Replace(filepath, "@", "", 1)
	content, err := ioutil.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("error reading payload file: %v\n", err)
		os.Exit(1)
	}
	return func(i int) string {
		return string(content)
	}
}

type Worker struct {
	WorkerId             int
	BrokerUrl            string
	Username             string
	Password             string
	SkipTLSVerification  bool
	NumberOfMessages     int
	PayloadGenerator     PayloadGenerator
	Timeout              time.Duration
	Retained             bool
	PublisherQoS         byte
	SubscriberQoS        byte
	CA                   []byte
	Cert                 []byte
	Key                  []byte
	PauseBetweenMessages time.Duration
}

func setSkipTLS(o *autopaho.ClientConfig) {
	oldTLSCfg := o.TlsCfg
	if oldTLSCfg == nil {
		oldTLSCfg = &tls.Config{}
	}
	oldTLSCfg.InsecureSkipVerify = true
	o.TlsCfg = oldTLSCfg
}

func NewTLSConfig(ca, certificate, privkey []byte) (*tls.Config, error) {
	// Import trusted certificates from CA
	certpool := x509.NewCertPool()
	ok := certpool.AppendCertsFromPEM(ca)

	if !ok {
		return nil, fmt.Errorf("CA is invalid")
	}

	// Import client certificate/key pair
	cert, err := tls.X509KeyPair(certificate, privkey)
	if err != nil {
		return nil, err
	}

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: false,
		// Certificates = list of certs client sends to server.
		Certificates: []tls.Certificate{cert},
	}, nil
}

func (w *Worker) Run(ctx context.Context) {
	verboseLogger.Printf("[%d] initializing\n", w.WorkerId)

	queue := make(chan [2]string)
	cid := w.WorkerId
	t := randomSource.Int31()

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	topicName := fmt.Sprintf(topicNameTemplate, hostname, w.WorkerId, t)
	subscriberClientId := fmt.Sprintf(subscriberClientIdTemplate, hostname, w.WorkerId, t)
	publisherClientId := fmt.Sprintf(publisherClientIdTemplate, hostname, w.WorkerId, t)

	burl, err := url.Parse(w.BrokerUrl)
	if err != nil {
		panic(err)
	}
	verboseLogger.Printf("[%d] topic=%s subscriberClientId=%s publisherClientId=%s\n", cid, topicName, subscriberClientId, publisherClientId)
	sc := autopaho.ClientConfig{
		BrokerUrls: []*url.URL{burl},
		KeepAlive:  uint16(w.Timeout.Seconds()),
		OnConnectionUp: func(cm *autopaho.ConnectionManager, _ *paho.Connack) {
			verboseLogger.Printf("[%d] subscribing to topic\n", w.WorkerId)
			if _, err := cm.Subscribe(ctx, &paho.Subscribe{Subscriptions: map[string]paho.SubscribeOptions{topicName: paho.SubscribeOptions{QoS: w.SubscriberQoS}}}); err != nil {
				resultChan <- Result{
					WorkerId:     w.WorkerId,
					Event:        SubscribeFailedEvent,
					Error:        true,
					ErrorMessage: err,
				}
				return
			}
		},
		OnConnectError: func(err error) {
			resultChan <- Result{
				WorkerId:     w.WorkerId,
				Event:        ConnectFailedEvent,
				Error:        true,
				ErrorMessage: fmt.Errorf("sub: %s", err),
			}
			verboseLogger.Printf("[%d] connecting subscriber\n", w.WorkerId)
		},
		Debug: paho.NOOPLogger{},
		ClientConfig: paho.ClientConfig{
			ClientID:      subscriberClientId,
			OnClientError: func(err error) { fmt.Printf("server requested disconnect: %s\n", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
			Router: paho.NewSingleHandlerRouter(func(msgpub *paho.Publish) {
				queue <- [2]string{msgpub.Topic, string(msgpub.Payload)}
			}),
		},
	}
	sc.SetUsernamePassword(w.Username, []byte(w.Password))
	if len(w.CA) > 0 || len(w.Key) > 0 {
		tlsConfig, err := NewTLSConfig(w.CA, w.Cert, w.Key)
		if err != nil {
			panic(err)
		}
		sc.TlsCfg = tlsConfig
	}

	if w.SkipTLSVerification {
		setSkipTLS(&sc)
	}

	sub, err := autopaho.NewConnection(ctx, sc)
	if err != nil {
		panic(err)
	}
	defer func() {
		verboseLogger.Printf("[%d] unsubscribe\n", w.WorkerId)

		if _, err := sub.Unsubscribe(ctx, &paho.Unsubscribe{Topics: []string{topicName}}); err != nil {
			fmt.Printf("failed to unsubscribe: %v\n", err)
		}

		//subscriber.Disconnect(5)
	}()

	sc.ClientConfig.ClientID = publisherClientId
	sc.OnConnectError = func(err error) {
		resultChan <- Result{
			WorkerId:     w.WorkerId,
			Event:        ConnectFailedEvent,
			Error:        true,
			ErrorMessage: fmt.Errorf("pub: %s", err),
		}
		verboseLogger.Printf("[%d] connecting publisher\n", w.WorkerId)
	}
	pub, err := autopaho.NewConnection(ctx, sc)
	if err != nil {
		panic(err)
	}
	verboseLogger.Printf("[%d] starting control loop %s\n", w.WorkerId, topicName)

	stopWorker := false
	receivedCount := 0
	publishedCount := 0
	pub.AwaitConnection(ctx)
	sub.AwaitConnection(ctx)
	t0 := time.Now()
	for i := 0; i < w.NumberOfMessages; i++ {
		text := w.PayloadGenerator(i)
		_, err = pub.Publish(ctx, &paho.Publish{Topic: topicName, QoS: w.PublisherQoS, Retain: w.Retained, Payload: []byte(text)})
		if err != nil {
			verboseLogger.Printf("[%d][%s] error publish: %s\n", w.WorkerId, topicName, err)
		}
		publishedCount++
		time.Sleep(w.PauseBetweenMessages)
	}
	//publisher.Disconnect(5)

	publishTime := time.Since(t0)
	verboseLogger.Printf("[%d] all messages published\n", w.WorkerId)

	t0 = time.Now()
	for receivedCount < w.NumberOfMessages && !stopWorker {
		select {
		case <-queue:
			receivedCount++

			verboseLogger.Printf("[%d] %d/%d received\n", w.WorkerId, receivedCount, w.NumberOfMessages)
			if receivedCount == w.NumberOfMessages {
				resultChan <- Result{
					WorkerId:          w.WorkerId,
					Event:             CompletedEvent,
					PublishTime:       publishTime,
					ReceiveTime:       time.Since(t0),
					MessagesReceived:  receivedCount,
					MessagesPublished: publishedCount,
				}
			} else {
				resultChan <- Result{
					WorkerId:          w.WorkerId,
					Event:             ProgressReportEvent,
					PublishTime:       publishTime,
					ReceiveTime:       time.Since(t0),
					MessagesReceived:  receivedCount,
					MessagesPublished: publishedCount,
				}
			}
		case <-ctx.Done():
			var event string
			var isError bool
			switch ctx.Err().(type) {
			case TimeoutError:
				verboseLogger.Printf("[%d] received abort signal due to test timeout", w.WorkerId)
				event = TimeoutExceededEvent
				isError = true
			default:
				verboseLogger.Printf("[%d] received abort signal", w.WorkerId)
				event = AbortedEvent
				isError = false
			}
			stopWorker = true
			resultChan <- Result{
				WorkerId:          w.WorkerId,
				Event:             event,
				PublishTime:       publishTime,
				MessagesReceived:  receivedCount,
				MessagesPublished: publishedCount,
				Error:             isError,
			}
		}
	}

	verboseLogger.Printf("[%d] worker finished\n", w.WorkerId)
}
