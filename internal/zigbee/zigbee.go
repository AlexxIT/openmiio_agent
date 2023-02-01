package zigbee

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"github.com/AlexxIT/openmiio_agent/pkg/serial"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os/exec"
	"strconv"
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
			baudRate = uint(i)
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
var z3app *exec.Cmd
var baudRate uint

func z3Worker(name string, arg ...string) {
	for {
		log.Debug().Msgf("[zigb] run %s", name)

		z3app = exec.Command(name, arg...)

		pipe, err := z3app.StdoutPipe()
		if err != nil {
			log.Panic().Err(err).Caller().Send()
		}

		if err = z3app.Start(); err != nil {
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

		if z3app == nil {
			break
		}
	}

	log.Debug().Msgf("[zigb] close %s", name)
}

func tcpWorker(addr, port string, hardware bool) {
	ln, err := net.Listen("tcp", ":"+addr)
	if err != nil {
		log.Panic().Err(err).Caller().Send()
	}

	var tcp net.Conn
	var ser io.ReadWriteCloser

	for {
		tcp, err = ln.Accept()
		if err != nil {
			log.Panic().Err(err).Caller().Send()
		}

		log.Debug().Stringer("addr", tcp.RemoteAddr()).Msg("[zigb] new connection")

		if z3app != nil {
			proc := z3app.Process
			z3app = nil
			_ = proc.Kill()
			_ = proc.Release()
		}

		ser, err = open(port, hardware)
		if err != nil {
			log.Warn().Err(err).Caller().Send()

			_ = tcp.Close()

			continue
		}

		go func() {
			b := make([]byte, 1024)
			for {
				n, err2 := ser.Read(b)
				if err2 != nil {
					if err2 == io.EOF {
						continue
					}
					log.Debug().Err(err2).Caller().Send()
					break
				}
				log.Trace().Msgf("[zigb] recv %x", b[:n])
				_, _ = tcp.Write(b[:n])
			}

			_ = tcp.Close()
			_ = ser.Close()
		}()

		b := make([]byte, 1024)
		for {
			n, err1 := tcp.Read(b)
			if err1 != nil {
				log.Debug().Err(err1).Caller().Send()
				break
			}
			log.Trace().Msgf("[zigb] send %x", b[:n])
			_, _ = ser.Write(b[:n])
		}

		_ = tcp.Close()
		_ = ser.Close()

		log.Debug().Stringer("addr", tcp.RemoteAddr()).Msg("[zigb] close connection")
	}
}

func open(port string, hardware bool) (io.ReadWriteCloser, error) {
	if baudRate == 0 {
		if probe(port, 115200, hardware) {
			baudRate = 115200
		} else if probe(port, 38400, hardware) {
			baudRate = 38400
		} else {
			return nil, errors.New("unknown baud rate")
		}
	}

	log.Debug().Msgf("[zigb] open %s baud=%d hw=%t", port, baudRate, hardware)

	return serial.Open(serial.OpenOptions{
		PortName:              port,
		BaudRate:              baudRate,
		DataBits:              8,
		StopBits:              1,
		RTSCTSFlowControl:     hardware,
		InterCharacterTimeout: 0, // timeout ms
		MinimumReadSize:       1,
	})
}

func probe(port string, baudRate uint, hardware bool) bool {
	conn, err := serial.Open(serial.OpenOptions{
		PortName:              port,
		BaudRate:              baudRate,
		DataBits:              8,
		StopBits:              1,
		RTSCTSFlowControl:     hardware,
		InterCharacterTimeout: 3000, // timeout ms
		MinimumReadSize:       0,
	})
	if err != nil {
		return false
	}

	// reset cmd
	_, _ = conn.Write([]byte{0x1A, 0xC0, 0x38, 0xBC, 0x7E})

	// collect 7 byte of answer
	b := make([]byte, 7)
	_, _ = conn.Read(b)

	// fix custom firmware first single byte
	if b[0] == 0x11 {
		_, _ = conn.Read(b)
	}

	_ = conn.Close()

	return bytes.Compare(b, []byte{0x1A, 0xC1, 0x02, 0x0B, 0x0A, 0x52, 0x7E}) == 0
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
