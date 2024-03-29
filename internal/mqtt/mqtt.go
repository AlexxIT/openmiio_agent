package mqtt

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/pkg/mqtt"
	proto "github.com/huin/mqtt"
	"github.com/rs/zerolog"
	"net"
	"os"
	"os/exec"
	"time"
)

func Init() {
	if !app.Enabled("mqtt") {
		return
	}

	log = app.GetLogger("mqtt")

	runPublic()

	if v, ok := app.Args["--mqtt.sub"]; ok {
		Subscribe(nil, v)
	}

	app.Publish = Publish

	go func() {
		for {
			worker()

			time.Sleep(time.Second * 10)
		}
	}()
}

type Handler func(topic string, payload []byte)

var conn *mqtt.ClientConn
var online bool

var tqs []proto.TopicQos
var handlers []Handler

var log zerolog.Logger

func runPublic() {
	// check if public mosquitto already running
	if err := exec.Command("sh", "-c", "netstat -ltnp | grep -q '0.0.0.0:1883'").Run(); err == nil {
		return
	}

	var cmd string

	if app.Model == app.ModelMGW && app.Firmware < "1.5.5" {
		// fix CPU 90% full time bug
		cmd = "killall mosquitto; sleep .5; mosquitto -d; sleep .5; killall zigbee_gw"
	} else {
		if err := fixMosquitto(); err != nil {
			log.Warn().Err(err).Caller().Send()
			return
		}
		cmd = "killall mosquitto; sleep .5; /tmp/mosquitto -d; sleep .5"
	}

	log.Info().Msg("[mqtt] run mosquitto on 0.0.0.0:1883")

	if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
		log.Warn().Err(err).Caller().Send()
	}
}

func fixMosquitto() error {
	data, err := os.ReadFile("/bin/mosquitto")
	if err != nil {
		return err
	}

	// 1. Fix for mosquitto v1 and v2, both MIPS and ARM
	//    Multimode Gateway fw < 1.5.5 uses mosquitto v1
	//    Multimode Gateway 2 fw < 1.0.6 uses mosquitto v1
	i := bytes.Index(data, []byte{'1', '2', '7', '.', '0', '.', '0', '.', '1', 0, 0, 0, 'l', 'o'})
	if i < 0 {
		return errors.New("unsupported mosquitto binary version")
	}

	copy(data[i:], []byte{'0', '.', '0', '.', '0', '.', '0', 0, 0, 0, 0, 0, 0, 0})

	// 2. Fix for mosquitto v2
	//    log("Warning: Interface %s does not support %s configuration.")
	//    return 6; => return 0;
	switch len(data) {
	case 387799: // mosquitto 2.0.15 for Multimode Gateway fw 1.5.5+
		data[0x26DD0] = 0
	case 231900: // mosquitto 2.0.15 for Multimode Gateway 2 fw 1.0.6+
		data[0x1853E] = 0
	}

	return os.WriteFile("/tmp/mosquitto", data, 0755)
}

func worker() {
	c, err := net.Dial("tcp", "127.0.0.1:1883")
	if err != nil {
		log.Warn().Err(err).Caller().Send()
		return
	}

	msg := &proto.Connect{
		ClientId:       "openmiio_agent",
		KeepAliveTimer: 60 * 60 * 18, // important for mosquitto v2
	}
	conn = mqtt.NewClientConn(c)
	if err = conn.Connect(msg); err != nil {
		log.Warn().Err(err).Caller().Send()
		return
	}

	log.Info().Msg("[mqtt] online")

	online = true

	conn.Subscribe(tqs)

	app.SendReport()

	for m := range conn.Incoming {
		log.Trace().Msgf("[mqtt] %s %s", m.TopicName, m.Payload)

		payload := m.Payload.(proto.BytesPayload)
		for _, handler := range handlers {
			handler(m.TopicName, payload)
		}
	}

	log.Info().Msg("[mqtt] offline")

	online = false
}

func Subscribe(handler Handler, topics ...string) {
	for _, topic := range topics {
		tqs = append(tqs, proto.TopicQos{Topic: topic})
	}
	if handler != nil {
		handlers = append(handlers, handler)
	}
}

func Publish(topic string, payload any, retain bool) {
	if !online {
		return
	}

	var b []byte

	switch payload.(type) {
	case []byte:
		b = payload.([]byte)
	case string:
		b = []byte(payload.(string))
	default:
		var err error
		if b, err = json.Marshal(payload); err != nil {
			log.Warn().Err(err).Caller().Send()
			return
		}
	}

	msg := &proto.Publish{
		Header:    proto.Header{Retain: retain},
		TopicName: topic,
		Payload:   proto.BytesPayload(b),
	}
	conn.Publish(msg)
}
