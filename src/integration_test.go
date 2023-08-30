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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	golog "log"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Jon-Bright/plantprism/device"
	"github.com/Jon-Bright/plantprism/logs"
	"github.com/Jon-Bright/plantprism/plant"
	pahopackets "github.com/eclipse/paho.mqtt.golang/packets"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
	"github.com/nsf/jsondiff"
)

const (
	DumpLocation      = "../dumps/"
	DumpAWSPort       = "8884"
	DumpDevice        = "a8d39911-7955-47d3-981b-fbd9d52f9221"
	ManualActionsFile = "test-manual-actions.json"
	DebugTSFmt        = "2006-01-02T15:04:05.999"
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
	err := plant.LoadPlants()
	if err != nil {
		log.Critical.Fatalf("Failed to load plants: %v", err)
	}
	ma, err := readManualActions()
	if err != nil {
		t.Fatalf("failed reading manual actions: %v", err)
	}
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
		err = processPCAP(t, DumpLocation+de.Name(), ma)
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

type manualActionTime time.Time

func (mt *manualActionTime) UnmarshalJSON(b []byte) error {
	var f float64
	err := json.Unmarshal(b, &f)
	if err != nil {
		return fmt.Errorf("failed manualActionTime unmarshal: %w", err)
	}
	sf, nsf := math.Modf(f)
	s := int64(sf)
	ns := int64(float64(time.Second) * nsf)
	*mt = manualActionTime(time.Unix(s, ns))
	return nil
}

type manualAction struct {
	Timestamp   manualActionTime `json:"ts"`
	Action      string
	MsgTopic    string
	Slot        string
	PlantID     plant.PlantID
	Regex       string
	Replacement string
}

func (ma manualAction) String() string {
	t := time.Time(ma.Timestamp)
	return fmt.Sprintf("[%s (%d): %s]", t.Local().Format(DebugTSFmt), t.Unix(), ma.Action)
}

type manualActions struct {
	l  []manualAction
	ix int
}

func readManualActions() (*manualActions, error) {
	aj, err := os.ReadFile(ManualActionsFile)
	if err != nil {
		return nil, fmt.Errorf("reading '%s' failed: %w", ManualActionsFile, err)
	}
	var ma []manualAction
	err = json.Unmarshal(aj, &ma)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling failed: %w", err)
	}
	lastT := time.Time{}
	for i, a := range ma {
		thisT := time.Time(a.Timestamp)
		if thisT.Before(lastT) {
			return nil, fmt.Errorf("time goes backward at ma %d, first: %v (%d), second: %v (%d)", i, lastT, lastT.Unix(), thisT, thisT.Unix())
		}
		lastT = thisT
	}
	return &manualActions{ma, 0}, nil
}

type dumpPacket struct {
	awsToPC   bool
	packetNum int
	ts        time.Time
	raw       []byte
	parsed    *pahopackets.PublishPacket
}

func (dp dumpPacket) String() string {
	var dir string
	if dp.awsToPC {
		dir = "A->P"
	} else {
		dir = "P->A"
	}
	return fmt.Sprintf("[%d: %s %s (%d)]", dp.packetNum, dir, dp.ts.Local().Format(DebugTSFmt), dp.ts.Unix())
}

func processPCAP(t *testing.T, name string, ma *manualActions) error {
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
	i := 1
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
			dp := dumpPacket{}

			// All our dump packets have AWS on port 8884
			// and the Plantcube on a random other port.
			dp.awsToPC = (tl.TransportFlow().Src().String() == DumpAWSPort)
			dp.packetNum = i
			dp.ts = p.Metadata().Timestamp
			dp.raw = app.Payload()

			err = processPayload(t, &dp, ma)
			if err != nil {
				return fmt.Errorf("packet %d payload error: %w", i, err)
			}
		}
		i++
	}
}

func processPayload(t *testing.T, dp *dumpPacket, ma *manualActions) error {
	for r := bytes.NewReader(dp.raw); r.Len() > 0; {
		cp, err := pahopackets.ReadPacket(r)
		if err != nil {
			return fmt.Errorf("ReadPacket: %w", err)
		}
		switch p := cp.(type) {
		case *pahopackets.PublishPacket:
			dp.parsed = p
			err = processPublish(t, dp, ma)
			if err != nil {
				return fmt.Errorf("processPublish: %w", err)
			}
		}
	}
	return nil
}

func processPublish(t *testing.T, dp *dumpPacket, ma *manualActions) error {

	for ; ma.ix < len(ma.l) && dp.ts.After(time.Time(ma.l[ma.ix].Timestamp)); ma.ix++ {
		ret, err := processManualAction(t, ma, dp)
		if err != nil {
			return fmt.Errorf("processing manualAction %d/%v with packet %v failed: %w", ma.ix, ma.l[ma.ix], dp, err)
		}
		if ret {
			ma.ix++
			return err
		}
	}

	// If the packet is from AWS to the Plantcube, then that's the
	// bit Plantprism is replacing. Expect it to send us that
	// packet. Otherwise, this is a Plantcube (or possibly app)
	// message that should be published to Plantprism.
	if dp.awsToPC {
		return expectFromPlantprism(t, dp)
	} else {
		return publishToPlantprism(t, dp)
	}
}

var pushed *dumpPacket

func processManualAction(t *testing.T, mas *manualActions, dp *dumpPacket) (bool, error) {
	ma := mas.l[mas.ix]
	d, err := device.Get(DumpDevice, nil)
	if err != nil {
		return false, fmt.Errorf("couldn't get device: %w", err)
	}
	switch ma.Action {
	case "ignore":
		if ma.MsgTopic != dp.parsed.TopicName {
			return false, fmt.Errorf("wrong topic, want '%s', got '%s'", ma.MsgTopic, dp.parsed.TopicName)
		}
		return true, nil
	case "replace":
		if ma.MsgTopic != dp.parsed.TopicName {
			return false, fmt.Errorf("wrong topic, want '%s', got '%s'", ma.MsgTopic, dp.parsed.TopicName)
		}
		re, err := regexp.Compile(ma.Regex)
		if err != nil {
			return false, fmt.Errorf("regexp '%s' compile failed: %w", ma.Regex, err)
		}
		dp.parsed.Payload = re.ReplaceAll(dp.parsed.Payload, []byte(ma.Replacement))
	case "push":
		if pushed != nil {
			return false, fmt.Errorf("packet %v already pushed", pushed)
		}
		if ma.MsgTopic != dp.parsed.TopicName {
			return false, fmt.Errorf("wrong topic, want '%s', got '%s'", ma.MsgTopic, dp.parsed.TopicName)
		}
		pushed = dp
		return true, nil
	case "pop":
		if pushed == nil {
			return false, fmt.Errorf("no packet pushed")
		}
		var err error
		err = processPublish(t, pushed, mas)
		if err != nil {
			return false, fmt.Errorf("popped packet error: %w", err)
		}
	case "bumpAWSVersion":
		d.AWSVersion++
	case "harvest":
		err = d.HarvestPlant(ma.Slot, time.Time(ma.Timestamp))
		if err != nil {
			return false, fmt.Errorf("harvest slot '%s' failed: %w", ma.Slot, err)
		}
	case "addPlant":
		err = d.AddPlant(ma.Slot, ma.PlantID, time.Time(ma.Timestamp))
		if err != nil {
			return false, fmt.Errorf("plant slot '%s', id '%d' failed: %w", ma.Slot, ma.PlantID, err)
		}
	case "defaultMode":
		err = d.SetMode(device.ModeDefault, time.Time(ma.Timestamp))
		if err != nil {
			return false, fmt.Errorf("default mode failed: %w", err)
		}
	case "silent":
		err = d.SetMode(device.ModeSilent, time.Time(ma.Timestamp))
		if err != nil {
			return false, fmt.Errorf("silent mode failed: %w", err)
		}
	case "cinema":
		err = d.SetMode(device.ModeCinema, time.Time(ma.Timestamp))
		if err != nil {
			return false, fmt.Errorf("cinema mode failed: %w", err)
		}
	default:
		return false, fmt.Errorf("unknown manual action '%s'", ma.Action)
	}
	return false, nil
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

func publishToPlantprism(t *testing.T, dp *dumpPacket) error {
	m := testMessage{
		t:       t,
		topic:   dp.parsed.TopicName,
		payload: dp.parsed.Payload,
	}
	messageHandlerWithTime(nil, &m, dp.ts)
	return nil
}

func expectFromPlantprism(t *testing.T, dp *dumpPacket) error {
	select {
	case m := <-testPub.msgs:
		compareMessages(t, dp, m)
	case <-time.After(time.Second * 2):
		t.Errorf("packet %v, timeout waiting for message %v", dp, dp.parsed)
	}
	return nil
}

func compareMessages(t *testing.T, dp *dumpPacket, m *pubMsg) {
	if m.topic != dp.parsed.TopicName {
		t.Errorf("packet %v: incorrect topic,\n got '%s', \nwant '%s'", dp, m.topic, dp.parsed.TopicName)
		return
	}
	if m.payload[0] == '{' && dp.parsed.Payload[0] == '{' {
		// Both payloads are JSON (the common case)
		opt := jsondiff.DefaultConsoleOptions()
		result, diff := jsondiff.Compare(m.payload, dp.parsed.Payload, &opt)
		if result != jsondiff.FullMatch {
			t.Errorf("packet %v: incorrect JSON payload, topic '%s', match result %s, diff '%s'", dp, m.topic, result, diff)
		}
	} else {
		// Something else, assume binary (probably a recipe)
		if !reflect.DeepEqual(m.payload, dp.parsed.Payload) {
			hexGot := hex.Dump(m.payload)
			hexWant := hex.Dump(dp.parsed.Payload)
			t.Errorf("packet %v: incorrect non-JSON payload, topic '%s',\n got '%s', \nwant '%s'", dp, m.topic, hexGot, hexWant)
		}
	}
}
