package gotun2socks

import (
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/missdeer/gotun2socks/internal/packet"
	"github.com/yinghuocho/gosocks"
)

const (
	MTU = 1500
)

var (
	localSocksDialer *gosocks.SocksDialer = &gosocks.SocksDialer{
		Auth:    &gosocks.AnonymousClientAuthenticator{},
		Timeout: 1 * time.Second,
	}

	_, ip1, _ = net.ParseCIDR("10.0.0.0/8")
	_, ip2, _ = net.ParseCIDR("172.16.0.0/12")
	_, ip3, _ = net.ParseCIDR("192.168.0.0/24")
)

type Tun2Socks struct {
	dev            io.ReadWriteCloser
	localSocksAddr string
	publicOnly     bool

	writerStopCh chan bool
	readerStopCh chan bool
	writeCh      chan interface{}

	tcpConnTrackLock sync.Mutex
	tcpConnTrackMap  map[string]*tcpConnTrack

	udpConnTrackLock sync.Mutex
	udpConnTrackMap  map[string]*udpConnTrack

	dnsServers []string
	cache      *dnsCache
}

func isPrivate(ip net.IP) bool {
	return ip1.Contains(ip) || ip2.Contains(ip) || ip3.Contains(ip)
}

func dialLocalSocks(localAddr string) (*gosocks.SocksConn, error) {
	return localSocksDialer.Dial(localAddr)
}

func New(dev io.ReadWriteCloser, localSocksAddr string, dnsServers []string, publicOnly bool, enableDnsCache bool) *Tun2Socks {
	t2s := &Tun2Socks{
		dev:             dev,
		localSocksAddr:  localSocksAddr,
		publicOnly:      publicOnly,
		writerStopCh:    make(chan bool, 10),
		readerStopCh:    make(chan bool, 10),
		writeCh:         make(chan interface{}, 10000),
		tcpConnTrackMap: make(map[string]*tcpConnTrack),
		udpConnTrackMap: make(map[string]*udpConnTrack),
		dnsServers:      dnsServers,
	}
	if enableDnsCache {
		t2s.cache = &dnsCache{
			storage: make(map[string]*dnsCacheEntry),
		}
	}
	return t2s
}

func (t2s *Tun2Socks) Stop() {
	t2s.readerStopCh <- true
	t2s.writerStopCh <- true

	t2s.dev.Close()

	t2s.tcpConnTrackLock.Lock()
	for _, tcpTrack := range t2s.tcpConnTrackMap {
		close(tcpTrack.quitByOther)
	}
	t2s.tcpConnTrackLock.Unlock()

	t2s.udpConnTrackLock.Lock()
	for _, udpTrack := range t2s.udpConnTrackMap {
		close(udpTrack.quitByOther)
	}
	t2s.udpConnTrackLock.Unlock()
}

func (t2s *Tun2Socks) Run() {
	// writer
	go func() {
		for {
			select {
			case pkt := <-t2s.writeCh:
				switch pkt.(type) {
				case *tcpPacket:
					tcp := pkt.(*tcpPacket)
					t2s.dev.Write(tcp.wire)
					releaseTCPPacket(tcp)
				case *udpPacket:
					udp := pkt.(*udpPacket)
					t2s.dev.Write(udp.wire)
					releaseUDPPacket(udp)
				}
			case <-t2s.writerStopCh:
				log.Printf("quit tun2socks writer")
				return
			}
		}
	}()

	// reader
	var buf [MTU]byte
	var ip packet.IPv4
	var tcp packet.TCP
	var udp packet.UDP
	for {
		select {
		case <-t2s.readerStopCh:
			log.Printf("quit tun2socks reader")
			return
		default:
			n, e := t2s.dev.Read(buf[:])
			if e != nil {
				// TODO: stop at critical error
				log.Printf("read packet error: %s", e)
				return
			}
			e = packet.ParseIPv4(buf[:n], &ip)
			if e != nil {
				log.Printf("error to parse IPv4: %s", e)
				continue
			}
			if t2s.publicOnly {
				if !ip.DstIP.IsGlobalUnicast() {
					continue
				}
				if isPrivate(ip.DstIP) {
					continue
				}
			}

			switch ip.Protocol {
			case packet.IPProtocolTCP:
				e = packet.ParseTCP(ip.Payload, &tcp)
				if e != nil {
					log.Printf("error to parse TCP: %s", e)
					continue
				}
				t2s.tcp(buf[:n], &ip, &tcp)

			case packet.IPProtocolUDP:
				e = packet.ParseUDP(ip.Payload, &udp)
				if e != nil {
					log.Printf("error to parse UDP: %s", e)
					continue
				}
				t2s.udp(buf[:n], &ip, &udp)

			default:
				// Unsupported packets
				log.Printf("Unsupported packet: protocol %d", ip.Protocol)
			}
		}
	}
}
