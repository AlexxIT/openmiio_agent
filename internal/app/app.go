package app

import (
	"bytes"
	"github.com/rs/zerolog"
	"os"
	"runtime"
	"strings"
)

var Version = "1.1.1"

func Init() {
	// init command arguments
	for _, key := range os.Args[1:] {
		var value string
		if i := strings.IndexByte(key, '='); i > 0 {
			key, value = key[:i], key[i+1:]
		}
		Args[key] = value
	}

	// init logs
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	writer := zerolog.ConsoleWriter{
		Out: os.Stdout, TimeFormat: "15:04:05.000", NoColor: true,
	}

	log = zerolog.New(writer).With().Timestamp().Logger().Level(zerolog.WarnLevel)
	log = GetLogger("level")

	// get device model and firmware version
	if b, err := os.ReadFile("/etc/build.prop"); err == nil {
		Firmware = getKey(b, "ro.sys.mi_fw_ver=") + "_" + getKey(b, "ro.sys.mi_build_num=")
		Model = getKey(b, "ro.sys.model=")
	} else if b, err = os.ReadFile("/etc/rootfs_fw_info"); err == nil {
		Firmware = getKey(b, "version=")
		Model = ModelMGW
	}

	log.Info().Msgf("openmiio_agent version %s %s/%s", Version, runtime.GOOS, runtime.GOARCH)
	log.Info().Msgf("init model=%s fw=%s", Model, Firmware)

	AddReport("openmiio", map[string]any{
		"version": Version,
		"uptime":  NewUptime(),
	})

	AddReport("gateway", map[string]any{
		"model":    Model,
		"firmware": Firmware,
	})

	AddReport("serial", SerialStats{})
}

func GetLogger(name string) zerolog.Logger {
	if s := Args["--log."+name]; s != "" {
		if lvl, err := zerolog.ParseLevel(s); err == nil {
			return log.Level(lvl)
		}
	}
	return log
}

func Enabled(name string) bool {
	_, ok := Args[name]
	return ok
}

const (
	ModelMGW   = "lumi.gateway.mgl03"
	ModelE1    = "lumi.gateway.aqcn02"
	ModelMGW2  = "lumi.gateway.mcn001"
	ModelM1S22 = "lumi.gateway.acn004"
)

var Firmware string
var Model string
var Args = map[string]string{}

var log zerolog.Logger

func getKey(b []byte, sub string) string {
	if i := bytes.Index(b, []byte(sub)); i > 0 {
		b = b[i+len(sub):]
	} else {
		return ""
	}
	if i := bytes.IndexByte(b, '\n'); i > 0 {
		return string(b[:i])
	}
	return string(b)
}
