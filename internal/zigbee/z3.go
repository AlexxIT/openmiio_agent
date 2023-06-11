package zigbee

import (
	"bufio"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"os/exec"
	"sync"
)

var z3cmd *exec.Cmd
var z3mu sync.Mutex
var z3stop bool

func z3kill() {
	z3mu.Lock()
	defer z3mu.Unlock()

	if z3stop {
		return
	}

	if z3cmd != nil {
		if err := z3cmd.Process.Kill(); err != nil {
			log.Warn().Err(err).Caller().Send()
		}
	}

	z3stop = true
}

func z3Worker(name string, arg ...string) {
	for {
		z3mu.Lock()

		if z3stop {
			z3mu.Unlock()
			break
		}

		log.Info().Str("app", name).Msg("[zigb] run")

		z3cmd = exec.Command(name, arg...)

		pipe, err := z3cmd.StdoutPipe()
		if err != nil {
			log.Fatal().Err(err).Caller().Send()
		}

		if err = z3cmd.Start(); err != nil {
			log.Fatal().Err(err).Caller().Send()
		}

		z3mu.Unlock()

		report.Z3Starts++
		report.Z3Uptime = app.NewUptime()

		r := bufio.NewReader(pipe)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}

			log.Trace().Msgf("[zigb] %s", line)

			if len(line) > 0 {
				mqtt.Publish("log/z3", line, false)
			}
		}

		_ = z3cmd.Wait()

		report.Z3Uptime = nil
	}

	log.Info().Str("app", name).Msg("[zigb] close")
}
