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

	if socketActive(OriginalSocket) {
		log.Info().Msg("[cent] using original socket")

		// move original socket to proxy socket place
		if err := os.Rename(OriginalSocket, ProxySocket); err != nil {
			log.Panic().Err(err).Caller().Send()
			return
		}
	} else if socketActive(ProxySocket) {
		log.Info().Msgf("[cent] using proxy socket")

		_ = os.Remove(OriginalSocket)
	} else {
		log.Warn().Msg("[cent] can't open socket")
		return
	}

	addr := &net.UnixAddr{Name: OriginalSocket, Net: "unixpacket"}
	ln, err := net.ListenUnix(addr.Net, addr)
	if err != nil {
		log.Fatal().Err(err).Caller().Send()
	}

	// force BT utility to reconnect to new socket
	_ = exec.Command("killall", "silabs_ncp_bt", "miio_bt").Run()

	for {
		// very important to use AcceptUnix vs Accept, because ARM linux can't handle KILL signal
		conn1, err := ln.AcceptUnix()
		if err != nil {
			log.Fatal().Err(err).Caller().Send()
		}

		conn2, err := net.Dial("unixpacket", ProxySocket)
		if err != nil {
			log.Fatal().Err(err).Caller().Send()
		}

		log.Info().Msg("[cent] accept conn")

		go proxy(conn1, conn2, true)
		go proxy(conn2, conn1, false)
	}
}

func socketActive(name string) bool {
	conn, err := net.Dial("unixpacket", name)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
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

	log.Info().Msgf("[cent] close conn")

	_ = conn1.Close()
	_ = conn2.Close()
}

var log zerolog.Logger
