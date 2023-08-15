package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/Jon-Bright/plantprism/logs"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type MQTT struct {
	c   paho.Client
	log *logs.Loggers
}

type brokerFlags struct {
	url         string
	username    string
	password    string
	clientID    string
	caCert      string
	keepAlive   time.Duration
	pingTimeout time.Duration
}

var bf brokerFlags

func InitFlags() {
	flag.StringVar(&bf.url, "broker_url", "ssl://localhost:8883", "MQTT broker's URL, including protocol and port")
	flag.StringVar(&bf.username, "broker_username", "", "Username for MQTT broker")
	flag.StringVar(&bf.password, "broker_password", "", "Password for MQTT broker")
	flag.StringVar(&bf.clientID, "broker_client_id", "", "Client ID for MQTT broker")
	flag.StringVar(&bf.caCert, "broker_ca_cert", "", "Filename of a custom CA cert to trust from the broker")
	flag.DurationVar(&bf.keepAlive, "broker_keep_alive", 60*time.Second, "Interval for sending keep-alive packets to the MQTT broker")
	flag.DurationVar(&bf.pingTimeout, "broker_ping_timeout", 130*time.Second, "Timeout after which the connection to the MQTT broker is regarded as dead")
}

func addCACert(opts *paho.ClientOptions, caCert string) (*paho.ClientOptions, error) {
	// Get the SystemCertPool, continue with an empty pool on error
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Read in the cert file
	certs, err := ioutil.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("failed to append %q to root CAs: %w", caCert, err)
	}

	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		paho.WARN.Println("No certs appended, using system certs only")
	}

	// Trust the augmented cert pool in our client
	config := &tls.Config{
		RootCAs: rootCAs,
	}
	return opts.SetTLSConfig(config), nil
}

func New(l *logs.Loggers, connectHandler paho.OnConnectHandler) (*MQTT, error) {
	paho.DEBUG = l.Info
	paho.WARN = l.Warn
	paho.ERROR = l.Error
	paho.CRITICAL = l.Critical

	opts := paho.NewClientOptions().
		AddBroker(bf.url).
		SetKeepAlive(bf.keepAlive).
		SetPingTimeout(bf.pingTimeout).
		SetOnConnectHandler(connectHandler)
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
		opts, err = addCACert(opts, bf.caCert)
		if err != nil {
			return nil, err
		}
	}

	c := paho.NewClient(opts)

	m := MQTT{c, l}
	return &m, nil
}

func (m *MQTT) Connect() error {
	if token := m.c.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (m *MQTT) Subscribe(topic string, handler paho.MessageHandler) error {
	token := m.c.Subscribe(topic, 1, handler)
	token.Wait()
	err := token.Error()
	if err != nil {
		return fmt.Errorf("subscribe failed for topic '%s': %w", topic, err)
	}
	m.log.Info.Printf("Subscribed to '%s'", topic)
	return nil
}
