package main

import (
	"flag"

	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/mqtt"
)

const (
	TOPIC_PREFIX_GRP = "Prefix"
	TOPIC_DEVICE_GRP = "Device"
	TOPIC_EVENT_GRP  = "Event"
	TOPIC_REGEX      = "^(?P<" + TOPIC_PREFIX_GRP + ">agl/prod|agl/all|\\$aws)/things/" + // Prefix
		"(?P<" + TOPIC_DEVICE_GRP + ">[[:xdigit:]]{8}(?:-[[:xdigit:]]{4}){3}-[[:xdigit:]]{12})/" + // Device UUID
		"(?P<" + TOPIC_EVENT_GRP + ">events/software/(info|warning)/put|mode|recipe/get|shadow/(get|update))$" // Actual event

	// MQTT's # wildcard must be at end of string and matches
	// anything following the specified prefix.  This is therefore
	// "all topics".
	TOPIC_WILDCARD = "#"
)

var log *logs.Loggers

func main() {
	mqtt.InitFlags()
	logName := flag.String("logfile", "plantprism.log", "Name of the log file to use")

	flag.Parse()

	log = logs.New(*logName)
	mq, err := mqtt.New(log)
	if err != nil {
		log.Critical.Fatalf("Unable to initialize MQTT: %v", err)
	}

	_ = mq
}
