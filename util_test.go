package main

import (
	"net/netip"
	"testing"

	"github.com/lainio/err2/try"
)

func TestParseIP(t *testing.T) {
	ip4 := try.To1(parseIP("192.168.4.1/24"))
	ip4.Compare(netip.MustParseAddr("192.168.4.1"))
	ip6 := try.To1(parseIP("fdd9:f800::2/24"))
	ip6.Compare(netip.MustParseAddr("fdd9:f800::2"))
}
