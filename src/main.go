package main

import (
	"flag"
	"regexp"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/mqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
)

const (
	TOPIC_PREFIX_GRP      = "Prefix"
	TOPIC_DEVICE_GRP      = "Device"
	TOPIC_EVENT_GRP       = "Event"
	TOPIC_DEVICE_ID_REGEX = "[[:xdigit:]]{8}(?:-[[:xdigit:]]{4}){3}-[[:xdigit:]]{12}"
	TOPIC_INCOMING_REGEX  = "^" +
		"(?P<" + TOPIC_PREFIX_GRP + ">agl/prod|agl/all|\\$aws)/things/" + // Prefix
		"(?P<" + TOPIC_DEVICE_GRP + ">" + TOPIC_DEVICE_ID_REGEX + ")/" + // Device UUID
		"(?P<" + TOPIC_EVENT_GRP + ">events/software/(info|warning)/put|mode|recipe/get|shadow/(get|update))" + // Actual event
		"$"
	TOPIC_OUTGOING_REGEX = "^" +
		"(agl/prod|agl/all|\\$aws)/things/" + TOPIC_DEVICE_ID_REGEX + "/" +
		"(shadow/get/accepted)$"

	// MQTT's # wildcard must be at end of string and matches
	// anything following the specified prefix.  This is therefore
	// "all topics".
	TOPIC_WILDCARD = "#"
)

var (
	log             *logs.Loggers
	mq              *mqtt.MQTT
	topicIncomingRe *regexp.Regexp
	topicOutgoingRe *regexp.Regexp
)

func connectHandler(c paho.Client) {
	log.Info.Printf("MQTT connected")
	err := mq.Subscribe(TOPIC_WILDCARD, messageHandler)
	if err != nil {
		log.Critical.Fatalf("Post-connect subscribe failed: %v", err)
	}
}

func messageHandler(c paho.Client, m paho.Message) {
	matches := topicIncomingRe.FindStringSubmatch(m.Topic())
	if matches == nil {
		if !topicOutgoingRe.MatchString(m.Topic()) {
			log.Error.Printf("Message topic '%s' unknown, ignoring", m.Topic())
			return
		}
		log.Info.Printf("Outgoing topic '%s' seen", m.Topic())
		return
	}
	prefix := matches[topicIncomingRe.SubexpIndex(TOPIC_PREFIX_GRP)]
	deviceID := matches[topicIncomingRe.SubexpIndex(TOPIC_DEVICE_GRP)]
	event := matches[topicIncomingRe.SubexpIndex(TOPIC_EVENT_GRP)]
	log.Info.Printf("Received message for device '%s', prefix '%s', event '%s'", deviceID, prefix, event)

	device, err := device.Get(deviceID, c)
	if err != nil {
		log.Error.Printf("Couldn't get device: %v", err)
		return
	}

	device.ProcessMessage(prefix, event, m.Payload())
}

func main() {
	device.InitFlags()
	mqtt.InitFlags()
	logName := flag.String("logfile", "plantprism.log", "Name of the log file to use")

	flag.Parse()

	log = logs.New(*logName)
	log.Info.Printf("Starting")

	device.SetLoggers(log)
	err := device.ProcessFlags()
	if err != nil {
		log.Critical.Fatalf("Device flags: %v", err)
	}
	topicIncomingRe = regexp.MustCompile(TOPIC_INCOMING_REGEX)
	topicOutgoingRe = regexp.MustCompile(TOPIC_OUTGOING_REGEX)

	mq, err = mqtt.New(log, connectHandler)
	if err != nil {
		log.Critical.Fatalf("Unable to initialize MQTT: %v", err)
	}
	err = mq.Connect()
	if err != nil {
		log.Critical.Fatalf("Unable to connect MQTT: %v", err)
	}

	select {}
}
