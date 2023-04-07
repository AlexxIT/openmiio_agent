package zigbee

import (
	"bufio"
	"bytes"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"github.com/AlexxIT/openmiio_agent/pkg/serial"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func Init() {
	z3 := app.Enabled("z3")
	tcp := app.Args["--zigbee.tcp"]

	if !z3 && tcp == "" {
		return
	}

	log = app.GetLogger("zigbee")

	switch app.Model {
	case app.ModelMGW:
		if app.Firmware <= "1.4.6" {
			log.Warn().Msgf("[zigb] firmware unsupported: %s", app.Firmware)
			return
		}

		preventRestart("Lumi_Z3GatewayHost_MQTT")
		_ = exec.Command("killall", "Lumi_Z3GatewayHost_MQTT").Run()
	case app.ModelE1, app.ModelMGW2:
		preventRestart("mZ3GatewayHost_MQTT")
		_ = exec.Command("killall", "mZ3GatewayHost_MQTT").Run()
	case app.ModelM1S22:
		log.Warn().Msgf("[zigb] M1S 2022 unsupported")
		return
	default:
		return
	}

	time.Sleep(time.Second)

	if z3 {
		switch app.Model {
		case app.ModelMGW:
			go z3Worker("Lumi_Z3GatewayHost_MQTT", "-n", "1", "-b", "115200", "-p", "/dev/ttyS2", "-d", "/data/silicon_zigbee_host/", "-r", "c")
		case app.ModelE1:
			go z3Worker("mZ3GatewayHost_MQTT", "-p", "/dev/ttyS1", "-d", "/data/")
		case app.ModelMGW2:
			go z3Worker("mZ3GatewayHost_MQTT", "-p", "/dev/ttyS1", "-d", "/data/zigbee_host/", "-r", "c")
		}
	}

	if tcp != "" {
		if s := app.Args["--zigbee.baud"]; s != "" {
			i, _ := strconv.Atoi(s)
			baudRate = uint32(i)
		}

		switch app.Model {
		case app.ModelMGW:
			go tcpWorker(tcp, "/dev/ttyS2", false)
		case app.ModelE1, app.ModelMGW2:
			go tcpWorker(tcp, "/dev/ttyS1", true)
		}
	}
}

var log zerolog.Logger
var baudRate uint32
var killZ3 func()

func z3Worker(name string, arg ...string) {
	runZ3 := true
	for runZ3 {
		log.Debug().Msgf("[zigb] run %s", name)

		z3 := exec.Command(name, arg...)

		killZ3 = func() {
			killZ3 = nil
			if err := z3.Process.Kill(); err != nil {
				log.Warn().Err(err).Caller().Send()
			}
			runZ3 = false
		}

		pipe, err := z3.StdoutPipe()
		if err != nil {
			log.Panic().Err(err).Caller().Send()
		}

		if err = z3.Start(); err != nil {
			log.Panic().Err(err).Caller().Send()
		}

		r := bufio.NewReader(pipe)
		for {
			var line []byte
			line, _, err = r.ReadLine()
			if err != nil {
				break
			}

			log.Trace().Msgf("[zigb] %s", line)

			mqtt.Publish("log/z3", line, false)
		}

		_ = z3.Wait()
	}

	log.Debug().Msgf("[zigb] close %s", name)
}

func tcpWorker(addr, port string, hardware bool) {
	ln, err := net.Listen("tcp", ":"+addr)
	if err != nil {
		log.Panic().Err(err).Caller().Send()
	}

	for {
		tcp, err := ln.Accept()
		if err != nil {
			log.Panic().Err(err).Caller().Send()
		}

		log.Debug().Stringer("addr", tcp.RemoteAddr()).Msg("[zigb] new connection")

		if killZ3 != nil {
			killZ3()
		}

		ser, err := open(port, hardware)
		if err != nil {
			log.Warn().Err(err).Caller().Send()

			_ = tcp.Close()

			continue
		}

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			b2 := make([]byte, 256)
			for {
				n2, err2 := ser.Read(b2)
				if err2 != nil {
					log.Debug().Err(err2).Caller().Send()
					break
				}
				if n2 <= 0 {
					continue
				}

				log.Trace().Msgf("[zigb] recv %x", b2[:n2])
				if _, err2 = tcp.Write(b2[:n2]); err2 != nil {
					log.Debug().Err(err2).Caller().Send()
					break
				}
			}

			wg.Done()
		}()

		b1 := make([]byte, 256)
		for {
			n1, err1 := tcp.Read(b1)
			if err1 != nil {
				log.Debug().Err(err1).Caller().Send()
				break
			}
			log.Trace().Msgf("[zigb] send %x", b1[:n1])
			if _, err1 = ser.Write(b1[:n1]); err1 != nil {
				log.Debug().Err(err1).Caller().Send()
				break
			}
		}

		_ = tcp.Close()
		_ = ser.Close()

		// wait until serial port will stop reading in separate gorutine
		wg.Wait()

		log.Debug().Stringer("addr", tcp.RemoteAddr()).Msg("[zigb] close connection")
	}
}

func open(port string, hardware bool) (io.ReadWriteCloser, error) {
	// custom zigbee firmware for Multimode Gateway work on 38400 speed
	if baudRate == 0 {
		if probe(port, 115200, hardware) {
			baudRate = 115200
		} else if probe(port, 38400, hardware) {
			baudRate = 38400
		} else {
			baudRate = 115200
			log.Warn().Msg("[zigb] fallback to default baud rate")
			//return nil, errors.New("unknown baud rate")
		}
	}

	return serial.Open(port, baudRate, hardware)
}

func probe(port string, baudRate uint32, hardware bool) bool {
	log.Trace().Msgf("[zigb] probe %s baud=%d hw=%t", port, baudRate, hardware)

	conn, err := serial.Open(port, baudRate, hardware)
	if err != nil {
		return false
	}

	defer func() {
		if err = conn.Close(); err != nil {
			log.Debug().Err(err).Caller().Send()
		}
	}()

	// reset cmd
	// https://www.silabs.com/documents/public/user-guides/ug101-uart-gateway-protocol-reference.pdf
	if _, err = conn.Write([]byte{0x1A, 0xC0, 0x38, 0xBC, 0x7E}); err != nil {
		return false
	}

	// important to use 2 second timeout on serial port, because chip reset takes 1 second
	b := make([]byte, 8)
	for n := 0; n < 8; {
		n1, err1 := conn.Read(b[n:])
		if err1 != nil {
			log.Debug().Err(err1).Caller().Send()
			return false
		}

		log.Trace().Msgf("[zigb] probe %x", b[n:n+n1])

		if n1 == 0 {
			return false
		}

		n += n1

		// right answer:  1a c1 02 0b 0a 52 7e
		// but sometimes: 11 1a c1 02 0b 0a 52 7e
		if bytes.Contains(b, []byte{0xC1, 0x02, 0x0B, 0x0A, 0x52}) {
			return true
		}
	}

	return false
}

// Hacky way of preventing program restarts:
// - `app` will print program name in the ps (so daemons won't restart it)
// - `tail -f /dev/null` will run forever
// - `Pdeathsig` will stop child if parent died (even with SIGKILL)
func preventRestart(app string) {
	// space after app name in ps is important!
	cmd := exec.Command("tail", app, "-f", "/dev/null")
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if err := cmd.Start(); err != nil {
		log.Panic().Err(err).Caller().Send()
	}
}
