package tests

import (
	"fmt"
	"testing"
)

func Assert(t *testing.T, v ...interface{}) {
	for i := range v {
		if v[i] != nil {
			v[i] = fmt.Sprintf("%+v", v[i])
		} else {
			v[i] = "<nil>"
		}
	}

	if len(v) == 1 {
		v = append(v, "true")
	}

	if v[0] != v[1] {
		t.Fatalf("%s != %s", v[0], v[1])
	}
}
