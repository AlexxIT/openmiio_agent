package app

import (
	"github.com/AlexxIT/openmiio_agent/pkg/tests"
	"testing"
)

func TestParseSerial(t *testing.T) {
	b := []byte(`serinfo:1.0 driver revision:
0: uart:16550A mmio:0x18147000 irq:17 tx:6337952 rx:0 RTS|CTS|DTR
1: uart:16550A mmio:0x18147400 irq:46 tx:294374665 rx:-1937325442 oe:1684 RTS|DTR
2: uart:16550A mmio:0x18147800 irq:47 tx:1846359 rx:3845724 oe:18 RTS|DTR
`)
	counters := parseSerial(b)
	tests.Assert(t, counters, []uint32{
		6337952, 0, 0,
		294374665, 2357641854, 1684,
		1846359, 3845724, 18,
	})
}
