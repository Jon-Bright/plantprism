package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
)

var (
	logInfo     *log.Logger
	logWarn     *log.Logger
	logError    *log.Logger
	logCritical *log.Logger
)

type brokerFlags struct {
	url         string
	username    string
	password    string
	clientID    string
	caCert      string
	keepAlive   time.Duration
	pingTimeout time.Duration
}

func initLogging(logName string) {
	l, err := os.OpenFile(logName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Unable to open log file: %v", err))
	}

	logInfo = log.New(l, "INFO: ", log.LstdFlags)
	logWarn = log.New(l, "WARN: ", log.LstdFlags)
	logError = log.New(l, "ERROR: ", log.LstdFlags)
	logCritical = log.New(l, "CRIT: ", log.LstdFlags)
}

func mqttAddCACert(opts *mqtt.ClientOptions, caCert string) (*mqtt.ClientOptions, error) {
	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Read in the cert file
	certs, err := ioutil.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("failed to append %q to root CAs: %v", caCert, err)
	}

	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		logWarn.Println("No certs appended, using system certs only")
	}

	// Trust the augmented cert pool in our client
	config := &tls.Config{
		RootCAs: rootCAs,
	}
	return opts.SetTLSConfig(config), nil
}

func mqttInit(bf *brokerFlags) (*mqtt.Client, error) {
	mqtt.DEBUG = logInfo
	mqtt.WARN = logWarn
	mqtt.ERROR = logError
	mqtt.CRITICAL = logCritical

	opts := mqtt.NewClientOptions().
		AddBroker(bf.url).
		SetKeepAlive(bf.keepAlive).
		SetPingTimeout(bf.pingTimeout)
	if bf.username != "" {
		opts = opts.SetUsername(bf.username)
	}
	if bf.password != "" {
		opts = opts.SetPassword(bf.password)
	}
	if bf.clientID != "" {
		opts = opts.SetClientID(bf.clientID)
	}
	if bf.caCert != "" {
		var err error
		opts, err = mqttAddCACert(opts, bf.caCert)
		if err != nil {
			return nil, err
		}
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &c, nil
}

func main() {
	bf := new(brokerFlags)
	flag.StringVar(&bf.url, "broker_url", "ssl://localhost:8883", "MQTT broker's URL, including protocol and port")
	flag.StringVar(&bf.username, "broker_username", "", "Username for MQTT broker")
	flag.StringVar(&bf.password, "broker_password", "", "Password for MQTT broker")
	flag.StringVar(&bf.clientID, "broker_client_id", "", "Client ID for MQTT broker")
	flag.StringVar(&bf.caCert, "broker_ca_cert", "", "Filename of a custom CA cert to trust from the broker")
	flag.DurationVar(&bf.keepAlive, "broker_keep_alive", 60*time.Second, "Interval for sending keep-alive packets to the MQTT broker")
	flag.DurationVar(&bf.pingTimeout, "broker_ping_timeout", 130*time.Second, "Timeout after which the connection to the MQTT broker is regarded as dead")

	logName := flag.String("logfile", "plantprism.log", "Name of the log file to use")

	flag.Parse()

	initLogging(*logName)
	mqttClient, err := mqttInit(bf)
	if err != nil {
		logCritical.Fatalf("Unable to initialize MQTT: %v", err)
	}
	_ = mqttClient
}
