package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	TOPIC_REGEX = "^(?P<Prefix>agl/prod|agl/all|\\$aws)/things/" + // Prefix
		"(?P<Device>[[:xdigit:]]{8}(?:-[[:xdigit:]]{4}){3}-[[:xdigit:]]{12})/" + // Device UUID
		"(?P<Event>events/software/(info|warning)/put|mode|recipe/get|shadow/(get|update))$" // Actual event

	// MQTT's # wildcard must be at end of string and matches
	// anything following the specified prefix.  This is therefore
	// "all topics".
	TOPIC_WILDCARD = "#"
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
	mq, err := NewMQTT(bf)
	if err != nil {
		logCritical.Fatalf("Unable to initialize MQTT: %v", err)
	}

	_ = mq
}
