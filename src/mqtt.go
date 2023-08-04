package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/eclipse/paho.mqtt.golang"
)

type MQTT struct {
	c *mqtt.Client
}

func addCACert(opts *mqtt.ClientOptions, caCert string) (*mqtt.ClientOptions, error) {
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

func NewMQTT(bf *brokerFlags) (*MQTT, error) {
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
		opts, err = addCACert(opts, bf.caCert)
		if err != nil {
			return nil, err
		}
	}

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	m := MQTT{&c}
	return &m, nil
}
