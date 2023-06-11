package miio

import (
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"github.com/rs/zerolog"
	"strconv"
)

const (
	AddrHardware   = int(1)   // basic_gw
	AddrBluetooth  = int(2)   // silabs_ncp_bt
	AddrZigbee     = int(4)   // zigbee_gw
	AddrHomeKit    = int(8)   // homekitserver
	AddrAutomation = int(16)  // mijia_automation
	AddrGateway    = int(32)  // basic_app
	AddrCentral    = int(128) // central_service_lite
	AddrCloud      = int(0)   // miio_client
	AddrMQQT       = int(-1)  // mosquitto
)

func appname(addr int) string {
	switch addr {
	case AddrHardware:
		return "hardware"
	case AddrBluetooth:
		return "bluetooth"
	case AddrZigbee:
		return "zigbee"
	case AddrHomeKit:
		return "homekit"
	case AddrAutomation:
		return "automation"
	case AddrGateway:
		return "gateway"
	case AddrCentral:
		return "central"
	case AddrCloud:
		return "cloud"
	case AddrMQQT:
		return "mqtt"
	}
	return strconv.Itoa(addr)
}

func Init() {
	if !app.Enabled("miio") {
		return
	}

	log = app.GetLogger("miio")

	mqtt.Subscribe(func(topic string, payload []byte) {
		if topic == "miio/command" {
			miioRequestRaw(AddrMQQT, payload)
		}
	}, "miio/command")

	app.AddReport("miio", &report)
	rpc.AddResponse(miioReport)

	go rpc.MarksWorker()

	go cloudWorker()
	go localWorker()
}

func Send(to int, b []byte) {
	switch to {
	case AddrCloud:
		sendToCloud(b)
	case AddrMQQT:
		mqtt.Publish("miio/command_ack", b, false)
	default:
		sendToUnicast(to, b)
	}
}

var log zerolog.Logger

func miioRequestRaw(from int, b []byte) {
	msg, err := rpc.NewMessage(b)
	if err != nil {
		log.Warn().Err(err).Caller().Msg(string(b))
		return
	}

	miioRequest(from, msg)
}

func miioRequest(from int, msg rpc.Message) {
	v, hasTO := msg["_to"]

	to := rpc.Atoi(v)
	if to > 0 {
		// 1. Request from local to multiple local (if to > 0)
		log.Trace().Msgf("[miio] %s msg from=%d to=%d", msg, from, to)

		if from == to {
			return // skip basic_gw bug
		}

		msg.SetInt("_from", from)
	} else if msg0, to0 := rpc.FindMessage(msg); msg0 != nil {
		// 2. Response from any to any (msg with original ID)
		log.Trace().Msgf("[miio] %s res from=%d to=%d", msg, from, to0)

		if from == AddrCloud && to0 > 0 && !hasPrefix(msg0["method"], `"local.`) {
			mqtt.Publish("miio/report_ack", msg, false)
		}

		miioResponse(to0, msg0, msg)
		return
	} else {
		// 3. Request from local to cloud (if from > 0)
		// 4. Request from cloud or mqtt to local (if from <= 0)
		// 5. Request from mqtt to cloud (if from=-1 and to=0)
		log.Trace().Msgf("[miio] %s req from=%d to=0", msg, from)

		if from > 0 && !hasPrefix(msg["method"], `"local.`) {
			mqtt.Publish("miio/report", msg, false)
		}
	}

	if hasTO {
		delete(msg, "_to")
	}

	for _, patch := range rpc.Requests {
		if patch(from, &msg) {
			log.Trace().Msgf("[miio] %s req patch", msg)
			break
		}
	}

	if msg == nil {
		return
	}

	if to == AddrCloud {
		// swap message ID, so we can catch response on this request
		rpc.MarkMessage(msg, from)
	}

	b, err := msg.Marshal()
	if err != nil {
		log.Warn().Err(err).Caller().Send()
		return
	}

	switch {
	case to > 0:
		// request from local to multiple local
		sendToMulticast(to, b)
	case from > 0 || hasTO:
		// request from local to cloud or from mqtt to cloud
		sendToCloud(b)
	default:
		// request from cloud or mqtt to local
		sendToMethod(msg.Method(), b)
	}
}

func miioResponse(to int, req, res rpc.Message) {
	for _, patch := range rpc.Responses {
		if patch(to, req, &res) {
			log.Trace().Msgf("[miio] %s res patch", res)
			break
		}
	}

	if res == nil {
		return
	}

	b, err := res.Marshal()
	if err != nil {
		log.Warn().Err(err).Caller().Send()
		return
	}

	Send(to, b)
}

func hasPrefix(b []byte, prefix string) bool {
	return len(b) >= len(prefix) && string(b[0:len(prefix)]) == prefix
}
