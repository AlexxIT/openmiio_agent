package cache

import (
	"bytes"
	"encoding/json"
	"github.com/AlexxIT/openmiio_agent/internal/store"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"hash/crc32"
)

func cacheBLEDevList(req, res rpc.Message) bool {
	var res1 struct {
		AuthedDevs []json.RawMessage `json:"authed_devs"`
		DeniedDevs json.RawMessage   `json:"denied_devs"`
	}

	if v, ok := res["result"]; ok {
		// split response on items
		if err := json.Unmarshal(v, &res1); err != nil {
			log.Warn().Err(err).Caller().Send()
			return false
		}

		var change bool
		for _, dev := range res1.AuthedDevs {
			key := crc32.Update(bleQueryDev, table, dev[:25])
			if c, hit := cache[key]; !hit || !bytes.Equal(c, dev) {
				cache[key] = dev
				change = true
			}
		}

		if change {
			_ = store.Set(cacheKey, &cache)
		}
	} else {
		// split request on items
		var req1 struct {
			Devices []json.RawMessage `json:"devices"`
		}
		if err := json.Unmarshal(req["params"], &req1); err != nil {
			log.Warn().Err(err).Caller().Send()
			return false
		}

		for _, dev := range req1.Devices {
			key := crc32.Update(bleQueryDev, table, dev[:25])
			if c, hit := cache[key]; hit {
				res1.AuthedDevs = append(res1.AuthedDevs, c)
			}
		}

		res1.DeniedDevs = []byte(`[]`)

		b, err := json.Marshal(res1)
		if err != nil {
			log.Warn().Err(err).Caller().Send()
			return false
		}

		res["result"] = b
		delete(res, "error")
		return true
	}

	return false
}
