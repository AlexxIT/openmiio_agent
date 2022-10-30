package miio

import (
	"bytes"
	"net"
	"time"
)

var conn0 net.Conn

func cloudWorker() {
	var err error
	var n int

	sep := []byte(`}{`)

	for {
		conn0, err = net.Dial("tcp", "localhost:54322")
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
