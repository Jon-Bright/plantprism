package main

import (
	"flag"
	"regexp"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/mqtt"
	"github.com/Jon-Bright/plantprism/plant"
	"github.com/Jon-Bright/plantprism/ui"
	"github.com/benbjohnson/clock"
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
		"(shadow/get/accepted|recipe)$"
)

var (
	log       *logs.Loggers
	mq        *mqtt.MQTT
	publisher device.Publisher

	topicIncomingRe = regexp.MustCompile(TOPIC_INCOMING_REGEX)
	topicOutgoingRe = regexp.MustCompile(TOPIC_OUTGOING_REGEX)

	// Mosquitto won't deliver topics that start with dollar signs
	// unless they're explicitly subscribed to - a wildcard
	// subscription is insufficient. Therefore, we subscribe to
	// them explicitly and use a wildcard for everything else.
	subTopics = []string{
		"#",
		"$aws/things/+/shadow/get",
		"$aws/things/+/shadow/update",
	}
)

func connectHandler(c paho.Client) {
	log.Info.Printf("MQTT connected")
	var (
		i     int
		topic string
	)
	mh := messageHandler
	for i, topic = range subTopics {
		err := mq.Subscribe(topic, mh)
		if err != nil {
			log.Critical.Fatalf("Post-connect subscribe for '%s' failed: %v", topic, err)
		}
		// We only want to set a message handler for the first
		// subscription (which is the wildcard
		// subscription). For the others, we do want to
		// subscribe, but we don't want that subscription to
		// call messageHandler - they'll match the wildcard at
		// _our_ end even if they don't at the broker's end.
		mh = nil
	}
	log.Info.Printf("Subscribed to %d topics", i)
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

	device, err := device.Get(deviceID, publisher)
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

	clk := clock.New()

	err := device.Init(log, clk)
	if err != nil {
		log.Critical.Fatalf("Device flags: %v", err)
	}
	err = plant.LoadPlants()
	if err != nil {
		log.Critical.Fatalf("Failed to load plants: %v", err)
	}

	mq, err = mqtt.New(log, connectHandler)
	if err != nil {
		log.Critical.Fatalf("Unable to initialize MQTT: %v", err)
	}
	publisher = mq
	ui.Init(log, mq)

	err = mq.Connect()
	if err != nil {
		log.Critical.Fatalf("Unable to connect MQTT: %v", err)
	}

	log.Info.Printf("Initialization complete")
	select {}
}
