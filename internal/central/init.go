package central

import (
	"bytes"
	"errors"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"github.com/rs/zerolog"
	"net"
	"os"
	"os/exec"
)

func Init() {
	if !app.Enabled("central") {
		return
	}

	log = app.GetLogger("central")

	const OriginalSocket = "/tmp/central_service_lite.socket"
	const ProxySocket = "/tmp/central_service_lite.proxy"

	// check if firmware works with central socket
	if _, err := os.Stat(OriginalSocket); errors.Is(err, os.ErrNotExist) {
		return
	}

	// move old socket to new socket place
	if _, err := os.Stat(ProxySocket); errors.Is(err, os.ErrNotExist) {
		log.Debug().Msgf("[cent] create proxy socket")

		if err = os.Rename(OriginalSocket, ProxySocket); err != nil {
			log.Error().Err(err).Caller().Send()
			return
		}
	} else {
		log.Debug().Msgf("[cent] use old proxy socket")

		_ = os.Remove(OriginalSocket)
	}

	sock, err := net.Listen("unixpacket", OriginalSocket)
	if err != nil {
		log.Panic().Err(err).Caller().Send()
	}

	// force BT utility to reconnect to new socket
	_ = exec.Command("killall", "silabs_ncp_bt", "miio_bt").Run()

	for {
		conn1, err := sock.Accept()
		if err != nil {
			panic(err)
		}

		conn2, err := net.Dial("unixpacket", ProxySocket)
		if err != nil {
			panic(err)
		}

		log.Debug().Msgf("[cent] new connection")

		go proxy(conn1, conn2, true)
		go proxy(conn2, conn1, false)
	}
}

func proxy(conn1, conn2 net.Conn, request bool) {
	var b = make([]byte, 4*1024)
	for {
		n, err := conn1.Read(b)
		if err != nil {
			break
		}

		if request {
			log.Trace().Msgf("[cent] %s req", b[:n])

			if !bytes.Contains(b[:n], []byte("local.get_gateway_role")) {
				// buffer may be overwriten before mqtt publish
				b2 := make([]byte, n)
				copy(b2, b)
				mqtt.Publish("central/report", b2, false)
			}
		} else {
			log.Trace().Msgf("[cent] %s res", b[:n])
		}

		if _, err = conn2.Write(b[:n]); err != nil {
			break
		}
	}

	log.Debug().Msgf("[cent] close connection")

	_ = conn1.Close()
	_ = conn2.Close()
}

var log zerolog.Logger
