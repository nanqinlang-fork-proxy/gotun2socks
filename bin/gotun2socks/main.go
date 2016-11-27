package main

import (
	"strings"

	"github.com/missdeer/gotun2socks"
	"github.com/missdeer/gotun2socks/tun"
	flag "github.com/ogier/pflag"
)

func main() {
	var tunDevice string
	var localSocksAddr string
	var dnsServers string
	var publicOnly bool
	var enableDnsCache bool
	flag.StringVar(&tunDevice, "tun-device", "tun0", "tun device name")
	flag.StringVar(&localSocksAddr, "local-socks-addr", "127.0.0.1:1080", "local SOCKS proxy address")
	flag.BoolVar(&publicOnly, "public-only", false, "only forward packets with public address destination")
	flag.StringVar(&dnsServers, "dns-server", "", "dns servers, enable quick release of DNS sessions, format: server1,server2...")
	flag.BoolVar(&enableDnsCache, "enable-dns-cache", false, "enable local dns cache if specified")
	flag.Parse()

	t := tun.New()
	t.Open()
	tun := gotun2socks.New(t.Fd, localSocksAddr, strings.Split(dnsServers, ","), publicOnly, enableDnsCache)
	tun.Run()
}
