package hms_test

import (
	"github.com/jordonwii/hms/hms"
	"testing"
)

func TestURLEncodeAndDecode(t *testing.T) {
	for i := 1; i < 100000; i++ {
		e := hms.ShortURLEncode(i)
		x := hms.ShortURLDecode(e)
		if x != i {
			t.Errorf("For i=%d, encoded to %s, but decoded to: %d", i, e, x)
			break
		}
	}
}
