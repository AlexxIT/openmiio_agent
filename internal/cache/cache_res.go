package cache

import (
	"bytes"
	"encoding/json"
	"github.com/AlexxIT/openmiio_agent/internal/store"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"hash/crc32"
)

const (
	bleQueryDev uint32 = iota + 1
	bleQueryProd
	homeroomInfo
)

func cacheResponse(key uint32, hash []byte, res rpc.Message) bool {
	if len(hash) > 0 {
		// crc32 simple, little and goroutine safe hash
		key = crc32.Update(key, table, hash)
	}

	c, hit := cache[key]

	if v, ok := res["result"]; ok {
		// if result exists
		if !hit || !bytes.Equal(c, v) {
			// if result not equal to cache
			cache[key] = v
			_ = store.Set(cacheKey, &cache)
		}
	} else {
		if hit {
			// if result in cache
			res["result"] = c
			delete(res, "error")
			return true
		}
	}

	return false // if result same
}

var cache = map[uint32]json.RawMessage{}

// we don't have arch for hardware IEEE or Castagnoli
// so using the best poly for small payloads
var table = crc32.MakeTable(crc32.Koopman)
