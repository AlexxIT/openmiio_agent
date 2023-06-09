package app

import (
	"fmt"
	"time"
)

var reports = map[string]any{}
var reportChan chan map[string]any
var reportTimer *time.Timer

func AddReport(name string, value any) {
	reports[name] = value
}

func GetReports() chan map[string]any {
	if reportChan != nil {
		log.Panic().Msg("multiple reports init")
	}

	reportChan = make(chan map[string]any)
	reportTimer = time.NewTimer(30 * time.Second)

	go func() {
		for range reportTimer.C {
			reportChan <- reports
			reportTimer.Reset(30 * time.Second)
		}
	}()

	return reportChan
}

func SendReport() {
	if reportTimer == nil {
		return
	}
	reportTimer.Reset(0)
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
