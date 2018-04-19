package endpoint

import (
	"fmt"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

const (
	mqttExpiresAfter = time.Second * 30
)

// MQTTConn is an endpoint connection
type MQTTConn struct {
	mu   sync.Mutex
	ep   Endpoint
	conn paho.Client
	ex   bool
	t    time.Time
}

// Expired returns true if the connection has expired
func (conn *MQTTConn) Expired() bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if !conn.ex {
		if time.Now().Sub(conn.t) > mqttExpiresAfter {
			conn.close()
			conn.ex = true
		}
	}
	return conn.ex
}

func (conn *MQTTConn) close() {
	if conn.conn != nil {
		if conn.conn.IsConnected() {
			conn.conn.Disconnect(250)
		}

		conn.conn = nil
	}
}

// Send sends a message
func (conn *MQTTConn) Send(msg string) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.ex {
		return errExpired
	}
	conn.t = time.Now()

	if conn.conn == nil {
		uri := fmt.Sprintf("tcp://%s:%d", conn.ep.MQTT.Host, conn.ep.MQTT.Port)
		ops := paho.NewClientOptions().SetClientID("tile38").AddBroker(uri)
		c := paho.NewClient(ops)

		if token := c.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}

		conn.conn = c
	}

	t := conn.conn.Publish(conn.ep.MQTT.QueueName, conn.ep.MQTT.Qos, conn.ep.MQTT.Retained, msg)
	t.Wait()

	if t.Error() != nil {
		conn.close()
		return t.Error()
	}

	return nil
}

func newMQTTConn(ep Endpoint) *MQTTConn {
	return &MQTTConn{
		ep: ep,
		t:  time.Now(),
	}
}
