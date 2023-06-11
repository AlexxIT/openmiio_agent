package app

import (
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"time"
)

func multiLogger(w io.Writer) io.Writer {
	if !Enabled("mqtt") {
		return w
	}

	reportTicker = time.NewTicker(30 * time.Second)

	go func() {
		for range reportTicker.C {
			Publish("openmiio/report", report, false)
		}
	}()

	return zerolog.MultiLevelWriter(w, mqttLogger{topic: "openmiio/log"})
}

var report = make(map[string]any, 9)
var reportTicker *time.Ticker

// Publish will be overwriten from MQTT module
var Publish = func(topic string, payload any, retain bool) {}

func AddReport(name string, value any) {
	report[name] = value
}

func SendReport() {
	if reportTicker == nil {
		return
	}
	reportTicker.Reset(30 * time.Second)
	Publish("openmiio/report", report, false)
}

type Uptime struct {
	start time.Time
}

func NewUptime() *Uptime {
	return &Uptime{start: time.Now()}
}

func (u *Uptime) MarshalJSON() ([]byte, error) {
	s := time.Since(u.start).Round(time.Second).String()
	return []byte(fmt.Sprintf(`"%s"`, s)), nil
}

type mqttLogger struct {
	topic string
}

func (w mqttLogger) WriteLevel(level zerolog.Level, p []byte) (n int, err error) {
	// it is important that `n` returns a length `p` or zerolor will print warning
	if n = len(p); n == 0 {
		return 0, nil
	}

	b := make([]byte, n-1) // remove tailing newline
	copy(b, p)

	Publish(w.topic, b, false)

	// wait msg will be published before app exit
	if level >= zerolog.FatalLevel {
		time.Sleep(100 * time.Millisecond)
	}

	return
}

func (w mqttLogger) Write(p []byte) (n int, err error) {
	// this function will never be called
	panic("not implemented")
}
