package hms_test

import (
	"github.com/jordonwii/hms/hms"
	"testing"
)

func testInt(t *testing.T, i int64) bool {
	e := hms.ShortURLEncode(i)
	t.Logf("For i=%d, encoded: %s", i, e)
	x := hms.ShortURLDecode(e)
	t.Logf("For i=%d, decoded: %d", i, x)
	if x != i {
		t.Errorf("For i=%d, encoded to %s, but decoded to: %d", i, e, x)
		return false
	}
	return true
}

func TestURLEncodeAndDecode(t *testing.T) {
	var i int64
	for i = 1; i < 100000; i++ {
		if !testInt(t, i) {
			break
		}
	}

	testInt(t, 4925812092436480)
}
