package cache

import (
	"bytes"
	"encoding/json"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
)

const (
	action       = 4097
	temperature  = 4100
	humidity     = 4102
	illumination = 4103
	moisture     = 4104
	conductivity = 4105
	battery      = 4106
	temperature2 = 4109
	lock         = 4110
	opening      = 4111
	idleTime     = 4119
	light        = 4120
)

func patchBLEProd(res rpc.Message) bool {
	v, ok := res["result"]
	if !ok {
		return false
	}

	var res1 struct {
		Operation string `json:"operation"`
		Pdid      uint16 `json:"pdid"`
		TTL       uint16 `json:"ttl"`
		Thr       int16  `json:"thr"`
		UpRule    []*struct {
			Delta float32 `json:"delta"`
			Eid   uint16  `json:"eid"`
			Intvl uint16  `json:"intvl"`
		} `json:"upRule"`
	}
	if err := json.Unmarshal(v, &res1); err != nil {
		return false
	}

	for _, rule := range res1.UpRule {
		switch rule.Eid {
		case 3, 4, 5, 6, 7, 8, 11, 15, action, idleTime, light:
			rule.Delta = 0 // sometimes 1
			rule.Intvl = 0 // sometimes 1
		case temperature, moisture, conductivity, temperature2:
			rule.Delta = 1   //original 1
			rule.Intvl = 180 // sometimes 600
		case humidity:
			rule.Delta = 5   // original 5
			rule.Intvl = 180 // sometimes 600
		case illumination, battery:
			rule.Delta = 0   // original 0
			rule.Intvl = 600 // original 600 or 1800
		case lock, opening:
			rule.Delta = 0
			rule.Intvl = 420
		}
	}

	b, err := json.Marshal(res1)
	if err != nil {
		return false
	}

	if !bytes.Equal(v, b) {
		res["result"] = b
		return true
	}

	return false
}
