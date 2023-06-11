package zigbee

import (
	"bytes"
	"errors"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/pkg/serial"
	"github.com/rs/zerolog"
	"io"
	"net"
	"os"
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

	app.AddReport("zigbee", &report)
}

var report struct {
	TcpRemote string      `json:"tcp_remote,omitempty"`
	TcpStarts int         `json:"tcp_starts,omitempty"`
	TcpUptime *app.Uptime `json:"tcp_uptime,omitempty"`
	Z3Starts  int         `json:"z3_starts,omitempty"`
	Z3Uptime  *app.Uptime `json:"z3_uptime,omitempty"`
}

var log zerolog.Logger
var baudRate uint32

func tcpWorker(addr, port string, hardware bool) {
	ln, err := net.Listen("tcp", ":"+addr)
	if err != nil {
		log.Fatal().Err(err).Caller().Send()
	}

	log.Info().Str("port", addr).Msg("[zigb] listen TCP")

	for {
		tcp, err := ln.Accept()
		if err != nil {
			log.Fatal().Err(err).Caller().Send()
		}

		z3kill()

		report.TcpRemote = tcp.RemoteAddr().String()
		report.TcpStarts++
		report.TcpUptime = app.NewUptime()

		log.Info().Str("remote", report.TcpRemote).Msg("[zigb] accept conn")

		ser, err := open(port, hardware)
		if err != nil {
			_ = tcp.Close()

			log.Fatal().Err(err).Str("port", port).Msg("[zigb] can't open serial")
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

		log.Info().Str("remote", report.TcpRemote).Msg("[zigb] close conn")

		report.TcpRemote = ""
		report.TcpUptime = nil
	}
}

func open(port string, hardware bool) (io.ReadWriteCloser, error) {
	// check if zigbee chip answer on reset command
	if baudRate == 0 {
		// Z3 app can accidentally disable the zigbee chip during reset process
		if err := zigbeeResetON(); err != nil {
			return nil, err
		}

		baudRate = 115200

		b, err := probe(port, hardware)
		if err != nil {
			log.Info().Err(err).Uint32("baud_rate", baudRate).Hex("read", b).Msg("[zigb] probe fail")

			if app.Model != app.ModelMGW {
				return nil, err
			}

			baudRate = 38400

			// custom zigbee firmware for Multimode Gateway work on 38400 speed
			b, err = probe(port, hardware)
			if err != nil {
				log.Info().Err(err).Uint32("baud_rate", baudRate).Hex("read", b).Msg("[zigb] probe fail")
				return nil, err
			}
		}

		log.Info().Uint32("baud_rate", baudRate).Hex("read", b).Msg("[zigb] probe OK")
	}

	return serial.Open(port, baudRate, hardware)
}

func probe(port string, hardware bool) ([]byte, error) {
	log.Debug().Str("port", port).Uint32("baud_rate", baudRate).Msg("[zigb] probe")

	conn, err := serial.Open(port, baudRate, hardware)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = conn.Close(); err != nil {
			log.Debug().Err(err).Caller().Send()
		}
	}()

	// reset cmd
	// https://www.silabs.com/documents/public/user-guides/ug101-uart-gateway-protocol-reference.pdf
	if _, err = conn.Write([]byte{0x1A, 0xC0, 0x38, 0xBC, 0x7E}); err != nil {
		return nil, err
	}

	// important to use 2 second timeout on serial port, because chip reset takes 1 second
	b := make([]byte, 32)
	for size := 0; size < len(b); {
		n, err := conn.Read(b[size:])
		if err != nil {
			return b[:size], err
		}

		log.Debug().Hex("read", b[size:size+n]).Msg("[zigb] probe")

		if n == 0 {
			return b[:size], err
		}

		size += n

		// right answer:  1a c1 02 0b 0a 52 7e
		// but sometimes: 11 1a c1 02 0b 0a 52 7e
		if bytes.Contains(b, []byte{0x1A, 0xC1, 0x02, 0x0B, 0x0A, 0x52, 0x7E}) {
			return b[:size], nil
		}
	}

	return b, errors.New("wrong response")
}

func zigbeeResetON() error {
	switch app.Model {
	case app.ModelMGW:
		// /bin/zigbee_inter_bootloader.sh 1
		// usleep(10000)
		// /bin/zigbee_reset.sh 0
		// usleep(10000)
		// /bin/zigbee_reset.sh 1
		return os.WriteFile("/sys/class/gpio/gpio18/value", []byte{'1'}, 0644)
	case app.ModelE1, app.ModelMGW2:
		// /bin/zigbee_isp.sh 1
		// usleep(10000)
		// /bin/zigbee_reset.sh 1
		// usleep(10000)
		// /bin/zigbee_reset.sh 0
		return os.WriteFile("/sys/class/gpio/gpio44/value", []byte{'0'}, 0644)
	}

	return nil
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
		log.Fatal().Err(err).Caller().Send()
	}
}
