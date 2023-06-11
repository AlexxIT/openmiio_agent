package main

import (
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"github.com/AlexxIT/openmiio_agent/internal/cache"
	"github.com/AlexxIT/openmiio_agent/internal/central"
	"github.com/AlexxIT/openmiio_agent/internal/lua"
	"github.com/AlexxIT/openmiio_agent/internal/miio"
	"github.com/AlexxIT/openmiio_agent/internal/mqtt"
	"github.com/AlexxIT/openmiio_agent/internal/store"
	"github.com/AlexxIT/openmiio_agent/internal/zigbee"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	app.Init() // before all

	store.Init()
	lua.Init() // before mqtt

	miio.Init()    // optional, before mqtt
	zigbee.Init()  // optional
	cache.Init()   // optional, after store
	mqtt.Init()    // optional
	central.Init() // optional

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Log().Stringer("signal", <-sigs).Msg("exit")
}
