package main

// This isn't a unit test, it's an integration test. It loads three things:
//
// * pcap files captured from real Plantcube MQTT comms
// * A JSON file representing the state of the Plantcube at the start of those captures
// * A JSON file representing app actions taken during the captures
//
// It uses all of these to replay what happened during the captures and test that
// Plantprism's output/reaction matches that of the real Agrilution/AWS service.

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	golog "log"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	pahopackets "github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
	"github.com/nsf/jsondiff"
)

const (
	DumpLocation = "../dumps/"
	DumpAWSPort  = "8884"
	DumpDevice   = "a8d39911-7955-47d3-981b-fbd9d52f9221"
)

var (
	testPub *testPublisher
)

func TestMain(m *testing.M) {
	device.InitFlags()
	flag.Set("device", DumpDevice)
	flag.Parse()
	os.Exit(m.Run())
}

func TestReplay(t *testing.T) {
	initLogging(t)
	initPublisher(t)
	device.SetTestMode()
	device.ProcessFlags()
	des, err := os.ReadDir(DumpLocation)
	if err != nil {
		t.Fatalf("failed reading dumps directory: %v", err)
	}
	sort.Slice(des, func(i, j int) bool {
		return des[i].Name() < des[j].Name()
	})
	for _, de := range des {
		if !strings.HasSuffix(de.Name(), ".pcapng") {
			continue
		}
		err = processPCAP(t, DumpLocation+de.Name())
		if err != nil {
			t.Fatalf("pcap processing of '%s' failed: %v", de.Name(), err)
		}
	}
}

type testLogWriter struct {
	t *testing.T
}

func (t *testLogWriter) Write(p []byte) (n int, err error) {
	t.t.Log(string(p))
	return len(p), nil
}

func initLogging(t *testing.T) {
	tlw := testLogWriter{t}
	testLog := golog.New(&tlw, "", golog.LstdFlags)
	log = &logs.Loggers{testLog, testLog, testLog, testLog}
	device.SetLoggers(log)
}

type pubMsg struct {
	topic   string
	payload []byte
}

type testPublisher struct {
	t    *testing.T
	msgs chan *pubMsg
}

func (tp *testPublisher) Publish(topic string, payload []byte) error {
	tp.msgs <- &pubMsg{topic, payload}
	return nil
}

func initPublisher(t *testing.T) {
	testPub = &testPublisher{
		t:    t,
		msgs: make(chan *pubMsg, 5),
	}

	// This is a global in main
	publisher = testPub
}

func processPCAP(t *testing.T, name string) error {
	t.Logf("Processing '%s'...", name)
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("unable to open: %w", err)
	}
	defer f.Close()

	r, err := pcapgo.NewNgReader(f, pcapgo.NgReaderOptions{
		// We get zero packets if we don't specify this,
		// probably(?) because the pcapng file (or a segment
		// within it?) is specifying ethernet link type and
		// our packets have LinuxSLL2 link type.
		WantMixedLinkType: true,
	})
	if err != nil {
		return fmt.Errorf("unable to create ng reader: %w", err)
	}

	ps := gopacket.NewPacketSource(r, layers.LinkTypeLinuxSLL2)
	i := 0
	for {
		p, err := ps.NextPacket()
		if err == io.EOF {
			t.Logf("'%s': complete after %d packets", name, i)
			return nil
		} else if err != nil {
			return fmt.Errorf("error on packet %d NextPacket: %w", i, err)
		}
		if el := p.ErrorLayer(); el != nil {
			return fmt.Errorf("packet %d decode error: %w", i, el.Error())
		}
		app := p.ApplicationLayer()
		if app != nil {
			tl := p.TransportLayer()
			if tl == nil {
				return fmt.Errorf("packet %d has application layer but no transport layer", i)
			}
			// All our dump packets have AWS on port 8884
			// and the Plantcube on a random other port.
			awsToPC := (tl.TransportFlow().Src().String() == DumpAWSPort)
			err = processPayload(t, awsToPC, i, p.Metadata().Timestamp, app.Payload())
			if err != nil {
				return fmt.Errorf("packet %d payload error: %w", i, err)
			}
		}
		i++
	}
}

func processPayload(t *testing.T, awsToPC bool, packetNum int, ts time.Time, payload []byte) error {
	for r := bytes.NewReader(payload); r.Len() > 0; {
		cp, err := pahopackets.ReadPacket(r)
		if err != nil {
			return fmt.Errorf("ReadPacket: %w", err)
		}
		switch p := cp.(type) {
		case *pahopackets.PublishPacket:
			err = processPublish(t, awsToPC, packetNum, ts, p)
			if err != nil {
				return fmt.Errorf("processPublish: %w", err)
			}
		}
	}
	return nil
}

func processPublish(t *testing.T, awsToPC bool, packetNum int, ts time.Time, p *pahopackets.PublishPacket) error {
	// If the packet is from AWS to the Plantcube, then that's the
	// bit Plantprism is replacing. Expect it to send us that
	// packet. Otherwise, this is a Plantcube (or possibly app)
	// message that should be published to Plantprism.
	if awsToPC {
		return expectFromPlantprism(t, packetNum, ts, p)
	} else {
		return publishToPlantprism(t, ts, p)
	}
}

type testMessage struct {
	t       *testing.T
	topic   string
	payload []byte
}

func (m *testMessage) Duplicate() bool {
	m.t.Fatalf("Unimplemented Duplicate called")
	return false
}

func (m *testMessage) Qos() byte {
	m.t.Fatalf("Unimplemented Qos called")
	return 0
}

func (m *testMessage) Retained() bool {
	m.t.Fatalf("Unimplemented Retained called")
	return false
}

func (m *testMessage) Topic() string {
	return m.topic
}

func (m *testMessage) MessageID() uint16 {
	m.t.Fatalf("Unimplemented MessageID called")
	return 0
}

func (m *testMessage) Payload() []byte {
	return m.payload
}

func (m *testMessage) Ack() {
	m.t.Fatalf("Unimplemented Ack called")
}

func publishToPlantprism(t *testing.T, ts time.Time, p *pahopackets.PublishPacket) error {
	m := testMessage{
		t:       t,
		topic:   p.TopicName,
		payload: p.Payload,
	}
	messageHandlerWithTime(nil, &m, ts)
	return nil
}

func expectFromPlantprism(t *testing.T, packetNum int, ts time.Time, p *pahopackets.PublishPacket) error {
	select {
	case m := <-testPub.msgs:
		compareMessages(t, packetNum, ts, m, p)
	case <-time.After(time.Second * 2):
		t.Errorf("packet %d, orig time %d, timeout waiting for message %v", packetNum, ts.Unix(), p)
	}
	return nil
}

func compareMessages(t *testing.T, packetNum int, ts time.Time, m *pubMsg, p *pahopackets.PublishPacket) {
	if m.topic != p.TopicName {
		t.Errorf("packet %d: incorrect topic,\n got '%s', \nwant '%s'", packetNum, m.topic, p.TopicName)
		return
	}
	if m.payload[0] == '{' && p.Payload[0] == '{' {
		// Both payloads are JSON (the common case)
		opt := jsondiff.DefaultConsoleOptions()
		result, diff := jsondiff.Compare(m.payload, p.Payload, &opt)
		if result != jsondiff.FullMatch {
			t.Errorf("packet %d: incorrect JSON payload, orig time %d, topic '%s', match result %s, diff '%s'", packetNum, ts.Unix(), m.topic, result, diff)
		}
	} else {
		// Something else, assume binary (probably a recipe)
		if !reflect.DeepEqual(m.payload, p.Payload) {
			hexGot := hex.Dump(m.payload)
			hexWant := hex.Dump(p.Payload)
			t.Errorf("packet %d: incorrect non-JSON payload, orig time %d, topic '%s',\n got '%s', \nwant '%s'", packetNum, ts.Unix(), m.topic, hexGot, hexWant)
		}
	}
}
