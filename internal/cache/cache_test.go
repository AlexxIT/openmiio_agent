package cache

import (
	"bytes"
	"encoding/json"
	"github.com/AlexxIT/openmiio_agent/internal/store"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"github.com/AlexxIT/openmiio_agent/pkg/tests"
	"testing"
)

func msg(s string) rpc.Message {
	if cache == nil {
		cache = map[uint32]json.RawMessage{}
	}

	m, err := rpc.NewMessage([]byte(s))
	if err != nil {
		panic(err)
	}
	return m
}

func TestCloud(t *testing.T) {
	req1 := msg(`{"id":123,"method":"local.query_status","params":""}`)
	res1 := msg(`{"id":123,"method":"local.status","params":"cloud_trying"}`)
	miioResponse(2, req1, &res1)
	tests.Assert(t, res1, `{"id":123,"method":"local.status","params":"cloud_connected"}`)
}

func TestQueryDev(t *testing.T) {
	req1 := msg(`{"id":123,"method":"_sync.ble_query_dev","params":{"mac":"AA:BB:CC:DD:EE:FF","pdid":1249}}`)
	res1 := msg(`{"id":123,"result":{"operation":"query_dev","mac":"AA:BB:CC:DD:EE:FF","ttl":1800,"pdid":1249,"did":"xxx","beaconkey":"xxx","token":"xxx","spec_supported":0}}`)
	miioResponse(2, req1, &res1)
	tests.Assert(t, bytes.Contains(store.Store[cacheKey], []byte("1073182952")))

	res2 := msg(`{"error":{"code":-30013,"message":"offline."},"id":123}`)
	miioResponse(2, req1, &res2)
	tests.Assert(t, res1, res2)
}

func TestQueryDevList(t *testing.T) {
	req1 := msg(`{"id":123,"method":"_sync.ble_query_dev_list","params":{"devices":[{"mac":"AA:BB:CC:DD:EE:FF","pdid":1249}]}}`)
	res1 := msg(`{"id":123,"result":{"authed_devs":[{"mac":"AA:BB:CC:DD:EE:FF","ttl":1800,"pdid":1249,"did":"xxx","beaconkey":"xxx","token":"xxx","spec_supported":0}],"denied_devs":[]}}`)
	miioResponse(2, req1, &res1)
	tests.Assert(t, bytes.Contains(store.Store[cacheKey], []byte("1073182952")))

	res2 := msg(`{"error":{"code":-30011,"message":"try out."},"id":123}`)
	miioResponse(2, req1, &res2)
	tests.Assert(t, res1, res2)
}

func TestHomeroomInfo(t *testing.T) {
	req1 := msg(`{"id":123,"method":"_sync.get_homeroom_info","params":{}}`)
	res1 := msg(`{"id":123,"result":{"uid":"xxx","homeid":"xxx"}}`)
	miioResponse(128, req1, &res1)
	tests.Assert(t, bytes.Contains(store.Store[cacheKey], []byte(`"3":`)))

	res2 := msg(`{"error":{"code":-30011,"message":"try out."},"id":123}`)
	miioResponse(128, req1, &res2)
	tests.Assert(t, res1, res2)
}

func TestBLEProd(t *testing.T) {
	raw1 := `{"id":123,"result":{"operation":"query_prod","pdid":2691,"ttl":1800,"thr":-40,"upRule":[{"delta":0,"eid":15,"intvl":0},{"delta":1,"eid":4119,"intvl":1},{"delta":0,"eid":4103,"intvl":600},{"delta":0,"eid":4106,"intvl":600},{"delta":1,"eid":4120,"intvl":1}]}}`
	raw2 := `{"id":123,"result":{"operation":"query_prod","pdid":2691,"ttl":1800,"thr":-40,"upRule":[{"delta":0,"eid":15,"intvl":0},{"delta":0,"eid":4119,"intvl":0},{"delta":0,"eid":4103,"intvl":600},{"delta":0,"eid":4106,"intvl":600},{"delta":0,"eid":4120,"intvl":0}]}}`
	req1 := msg(`{"id":123,"method":"_sync.ble_query_prod","params":{"pdid":2691}}`)
	res1 := msg(raw1)
	miioResponse(2, req1, &res1)
	tests.Assert(t, res1, raw2)
}
