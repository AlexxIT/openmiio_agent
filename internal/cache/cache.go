package cache

import (
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/miio"
	"github.com/AlexxIT/openmiio_agent/internal/store"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"github.com/rs/zerolog"
)

const cacheKey = "cache"

func Init() {
	if !app.Enabled("cache") {
		return
	}

	log = app.GetLogger("cache")

	if err := store.Get(cacheKey, &cache); err != nil {
		log.Warn().Err(err).Caller().Send()
	}

	rpc.AddResponse(miioResponse)
}

var log zerolog.Logger

func miioResponse(to int, req rpc.Message, res *rpc.Message) bool {
	switch to {
	case miio.AddrBluetooth:
		switch string(req["method"]) {
		case `"local.query_status"`:
			// fix bluetooth without cloud connection
			switch string((*res)["params"]) {
			case `"cloud_trying"`, `"cloud_retry"`:
				(*res)["params"] = []byte(`"cloud_connected"`)
				return true
			}

		case `"_sync.ble_query_dev"`: // fw < 1.5.4
			// take only MAC part from params
			return cacheResponse(bleQueryDev, req["params"][:25], *res)

		case `"_sync.ble_query_dev_list"`: // fw >= 1.5.4
			return cacheBLEDevList(req, *res)

		case `"_sync.ble_query_prod"`:
			ok := cacheResponse(bleQueryProd, req["params"], *res)
			return ok || patchBLEProd(*res)

		case `"_async.ble_event"`:
			if _, err := (*res)["error"]; err {
				// not really necessary, but anyway
				(*res)["result"] = []byte(`"ok"`)
				delete(*res, "error")
				return true
			}

		case `"_sync.ble_keep_alive"`:
			if _, err := (*res)["error"]; err {
				(*res)["result"] = []byte(`{"operation":"keep_alive","intvl":1200,"delta":300,"filter":[]}`)
				delete(*res, "error")
				return true
			}
		}

	case miio.AddrCentral:
		switch string(req["method"]) {
		case `"_sync.get_homeroom_info"`:
			return cacheResponse(homeroomInfo, nil, *res)
		}
	}

	return false
}
