package miio

import (
	"encoding/json"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
)

var report struct {
	CloudStarts int             `json:"cloud_starts,omitempty"`
	CloudState  json.RawMessage `json:"cloud_state,omitempty"`
	CloudUptime *app.Uptime     `json:"cloud_uptime,omitempty"`
}

var cloudState string

func miioReport(to int, req rpc.Message, res *rpc.Message) bool {
	if string(req["method"]) == `"local.query_status"` {
		if state := string((*res)["params"]); state != cloudState {
			cloudState = state

			// params is bytes slice with quotes
			report.CloudState = (*res)["params"]

			if state == `"cloud_connected"` {
				report.CloudStarts++
				report.CloudUptime = app.NewUptime()
			} else {
				report.CloudState = nil
			}
		}
	}

	return false // because we don't change response
}
