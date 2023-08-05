package main

import (
	"flag"

	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/mqtt"
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
