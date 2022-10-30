package lua

import (
	"errors"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/miio"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"github.com/AlexxIT/openmiio_agent/pkg/rpc"
	"github.com/layeh/gopher-json"
	"github.com/rs/zerolog"
	"github.com/yuin/gopher-lua"
	"os"
	"sync"
)

func Init() {
	name, _ := os.Executable()
	name += ".lua"

	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		return
	}

	log = app.GetLogger("lua")

	L = lua.NewState()

	// add json.decode/json.encode functions
	L.PreloadModule("json", json.Loader)

	// add global mosquitto_pub function
	L.SetGlobal("mosquitto_pub", L.NewFunction(MosquittoPub))

	L.SetGlobal("miio_send", L.NewFunction(MiioSend))

	if err := L.DoFile(name); err != nil {
		log.Warn().Err(err).Caller().Send()
		return
	}

	if v := L.GetGlobal("miio_request"); v != lua.LNil {
		rpc.AddRequest(miioRequest(v))
	}

	if v := L.GetGlobal("miio_response"); v != lua.LNil {
		rpc.AddResponse(miioResponse(v))
	}

	if v := L.GetGlobal("mosquitto_sub"); v != lua.LNil {
		mqtt.Subscribe(func(topic string, payload []byte) {
			mu.Lock()

			L.Push(v)
			L.Push(lua.LString(topic))
			L.Push(lua.LString(payload))
			L.Call(2, 0)

			mu.Unlock()
		}, "#")
	}
}

func miioRequest(f lua.LValue) rpc.Request {
	return func(from int, req *rpc.Message) bool {
		method := req.Method()
		b, _ := req.Marshal()

		mu.Lock()

		L.Push(f)
		L.Push(lua.LNumber(from))
		L.Push(lua.LString(method))
		L.Push(lua.LString(b))
		L.Call(3, lua.MultRet)

		top := L.GetTop()
		if top == 0 {
			mu.Unlock()
			return false
		}

		result := L.Get(1)
		L.Pop(top)

		mu.Unlock()

		if result == lua.LNil {
			*req = nil
			return true
		}

		s := result.String()

		log.Trace().Msgf("[lua]  %s miio_request", s)

		*req, _ = rpc.NewMessage([]byte(s))
		return true
	}
}

func miioResponse(f lua.LValue) rpc.Response {
	return func(to int, req rpc.Message, res *rpc.Message) bool {
		method := req.Method()
		b0, _ := req.Marshal()
		b1, _ := res.Marshal()

		mu.Lock()

		L.Push(f)
		L.Push(lua.LNumber(to))
		L.Push(lua.LString(method))
		L.Push(lua.LString(b0))
		L.Push(lua.LString(b1))
		L.Call(4, lua.MultRet)

		top := L.GetTop()
		if top == 0 {
			mu.Unlock()
			return false
		}

		result := L.Get(1)
		L.Pop(top)

		mu.Unlock()

		if result == lua.LNil {
			*res = nil
			return true
		}

		s := result.String()

		log.Trace().Msgf("[lua]  %s miio_response", s)

		*res, _ = rpc.NewMessage([]byte(s))
		return true
	}
}

func MosquittoPub(L *lua.LState) int {
	topic := L.ToString(1)
	payload := L.ToString(2)
	retain := L.ToBool(3)

	mqtt.Publish(topic, payload, retain)

	return 0
}

func MiioSend(L *lua.LState) int {
	to := L.ToInt(1)
	payload := L.ToString(2)

	log.Trace().Msgf("[lua]  %s miio_send to=%d", payload, to)

	miio.Send(to, []byte(payload))

	return 0
}

var L *lua.LState
var mu sync.Mutex
var log zerolog.Logger
