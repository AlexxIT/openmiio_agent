package miio

import (
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"bytes"
	"net"
	"time"
)

var conn0 net.Conn

func cloudWorker() {
	var err error
	var n int
	var connection string

	sep := []byte(`}{`)

	for {
		switch app.Model {
		case app.ModelM2, app.ModelM1S, app.ModelM2PoE, app.ModelG3, app.ModelM3:
			connection = "127.0.0.1:21397"
		default:
			connection = "localhost:54322"
		}

		conn0, err = net.Dial("tcp", connection)
		if err != nil {
			time.Sleep(time.Second * 10)
			continue
		}

		log.Info().Msg("[miio] connected to miio_client")

		buf := make([]byte, 4096)
		for {
			n, err = conn0.Read(buf)
			if err != nil {
				break
			}

			// split multiple JSON in one packet
			b := buf[:n]
			for {
				if i := bytes.Index(b, sep); i > 0 {
					miioRequestRaw(AddrCloud, b[:i+1])
					b = b[i+1:]
					continue
				}
				miioRequestRaw(AddrCloud, b)
				break
			}
		}

		conn0 = nil
	}
}

func sendToCloud(b []byte) {
	if conn0 != nil {
		if _, err := conn0.Write(b); err != nil {
			log.Warn().Err(err).Caller().Send()
		}
	}
}
