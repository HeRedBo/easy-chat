package bitmap

import (
	"testing"

	"github.com/gookit/goutil/dump"
)

func TestBitmap_Set(t *testing.T) {
	b := NewBitmap(5)

	b.Set("pppp")
	b.Set("222")
	b.Set("ccc")
	b.Set("eee")
	b.Set("fff")
	dump.P(b.IsSet("222"))
	dump.P(b.Count())
	for _, bit := range b.bits {
		t.Logf("%b, %v", bit, bit)
	}
}
