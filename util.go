package main

import (
	"fmt"
	"net"
	"net/netip"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
)

func parseIP(s string) (addr netip.Addr, err error) {
	var ip net.IP
	if ip, _, err = net.ParseCIDR(s); err == nil {
		if ip.To4() != nil {
			ip = ip.To4()
		}
	} else if ip = net.ParseIP(s); ip == nil {
		return netip.Addr{}, fmt.Errorf("ip parse failed")
	}
	addr, _ = netip.AddrFromSlice(ip)
	return
}

func convertToFullAddr(NICID tcpip.NICID, endpoint netip.AddrPort) (tcpip.FullAddress, tcpip.NetworkProtocolNumber) {
	var protoNumber tcpip.NetworkProtocolNumber
	if endpoint.Addr().Is4() {
		protoNumber = ipv4.ProtocolNumber
	} else {
		protoNumber = ipv6.ProtocolNumber
	}
	return tcpip.FullAddress{
		NIC:  NICID,
		Addr: tcpip.Address(endpoint.Addr().AsSlice()),
		Port: endpoint.Port(),
	}, protoNumber
}
