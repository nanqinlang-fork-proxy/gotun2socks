# gotun2socks

A Golang implementation of tun2socks

[![Build Status](https://secure.travis-ci.org/missdeer/gotun2socks.png)](https://travis-ci.org/missdeer/gotun2socks)

[![wercker status](https://app.wercker.com/status/6c43597eba8359a74451ede438e98a31/m/master "wercker status")](https://app.wercker.com/project/byKey/6c43597eba8359a74451ede438e98a31)

## Usage

See <a href="https://code.google.com/p/badvpn/wiki/tun2socks"> Tun2Socks Introduction </a> for how to create tun device and change routing table.

## UDP forwarding

This implementation forwards UDP using standard SOCKS5 UDP request/reply. Thus to make UDP-based protocols (such as DNS) work, it needs to be chained with a UDP-enabled SOCKS5 proxy.  


## Credits

- https://github.com/google/gopacket
- https://github.com/ambrop72/badvpn/
- https://github.com/songgao/water
