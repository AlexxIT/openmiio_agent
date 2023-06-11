package miio

import (
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

func localWorker() {
	_ = os.Remove("/tmp/miio_agent.socket")
	_ = exec.Command("killall", "miio_agent").Run()

	// fix basic_gw (Multimode Gateway) bug with instant reconnection
	time.Sleep(time.Millisecond * 500)

	sock, err := net.Listen("unixpacket", "/tmp/miio_agent.socket")
	if err != nil {
		log.Fatal().Err(err).Caller().Send()
	}

	for {
		var conn net.Conn
		conn, err = sock.Accept()
		if err != nil {
			log.Warn().Err(err).Caller().Send()
			continue
		}

		go localClientWorker(conn)
	}
}

func localClientWorker(conn net.Conn) {
	var from int

	b := make([]byte, 4096)
	for {
		n, err := conn.Read(b)
		if err != nil {
			break
		}

		msg, err := rpc.NewMessage(b[:n])
		if err != nil {
			log.Warn().Err(err).Caller().Send()
			continue
		}

		if from == 0 {
			if string(msg["method"]) == `"bind"` {
				if from = msg.GetInt("address"); from > 0 {
					log.Info().Str("app", appname(from)).Msg("[miio] accept conn")

					mu.Lock()
					conns[from] = conn
					mu.Unlock()
				}
			}
			continue
		}

		if string(msg["method"]) == `"register"` {
			log.Trace().Msgf("[miio] %s addr=%d", b[:n], from)

			if key := msg.GetString("key"); key != "" {
				mu.Lock()
				methods[key] = append(methods[key], from)
				mu.Unlock()
			}
			continue
		}

		miioRequest(from, msg)
	}

	if from > 0 {
		log.Info().Str("app", appname(from)).Msg("[miio] close conn")

		mu.Lock()
		delete(conns, from)
		mu.Unlock()
	}
}

var mu sync.RWMutex
var conns = map[int]net.Conn{}
var methods = map[string][]int{}

func sendToMulticast(to int, b []byte) {
	mu.RLock()
	for addr, conn := range conns {
		if addr&to > 0 {
			if _, err := conn.Write(b); err != nil {
				log.Warn().Err(err).Caller().Send()
			}
		}
	}
	mu.RUnlock()
}

func sendToUnicast(to int, b []byte) {
	mu.RLock()
	if conn, ok := conns[to]; ok {
		if _, err := conn.Write(b); err != nil {
			log.Warn().Err(err).Caller().Send()
		}
	}
	mu.RUnlock()
}

func sendToMethod(method string, b []byte) {
	mu.RLock()
	for localMethod, addrs := range methods {
		if method != localMethod {
			continue
		}
		for _, to := range addrs {
			if conn, ok := conns[to]; ok {
				if _, err := conn.Write(b); err != nil {
					log.Warn().Err(err).Caller().Send()
				}
			}
		}
	}
	mu.RUnlock()
}
