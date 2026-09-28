package app

import "testing"

func TestConversions(t *testing.T) {
	c := 28.5
	f := cToF(c)
	k := cToK(c)
	if int(f*10) != int((c*1.8+32)*10) {
		t.Fatalf("unexpected F: %v", f)
	}
	if int(k) != int(c+273) {
		t.Fatalf("unexpected K: %v", k)
	}
}
