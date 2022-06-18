package glink

import (
	"net"
	"sync"
	"time"

	"github.com/juju/loggo"
)

const (
	srvAddr         = "224.0.0.1:9999"
	maxDatagramSize = 8192
)

var knownNodes = &sync.Map{}

type DiscoveryInfo struct {
	ClientId   string
	ClientName string
	Endpoint   string
}

type Discovery struct {
	NewNodes chan DiscoveryInfo
	OwnInfo  NodeAnnounce
	log      *loggo.Logger
}

func NewDiscovery(own_info NodeAnnounce, log *loggo.Logger) *Discovery {
	return &Discovery{NewNodes: make(chan DiscoveryInfo), OwnInfo: own_info, log: log}
}

func (d *Discovery) Run() {
	go d.serve()
	go d.ping()
}

func (*Discovery) Close() {

}

func (d *Discovery) serve() {
	addr, err := net.ResolveUDPAddr("udp", srvAddr)
	if err != nil {
		d.log.Errorf("%w", err)
	}

	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		d.log.Errorf("Cannot listen multicast", err)
	}
	l.SetReadBuffer(maxDatagramSize)
	for {
		buffer := make([]byte, maxDatagramSize)
		n, src, err := l.ReadFromUDP(buffer)
		if err != nil || n == 0 {
			d.log.Errorf("ReadFromUDP failed:", err)
			continue
		}

		hdr, err := DecodeHeader(buffer)
		if err != nil {
			d.log.Errorf("Cannot decode header:", err)
			continue
		}

		payload := buffer[6 : 6+hdr.PayloadSize]
		msg, err := DecodeMsg[NodeAnnounce](payload)
		if err != nil {
			d.log.Errorf("Cannot decode payload:", payload)
			continue
		}

		if _, has := knownNodes.Load(src.String()); !has {
			knownNodes.Store(src.String(), nil)
			d.NewNodes <- DiscoveryInfo{ClientId: msg.Cid, ClientName: msg.Name, Endpoint: msg.Endpoint}
		}
	}
}

func (d *Discovery) ping() {
	addr, err := net.ResolveUDPAddr("udp", srvAddr)
	if err != nil {
		d.log.Errorf("%w", err)
	}
	c, err := net.DialUDP("udp", nil, addr)
	msg, err := EncodeMsg(d.OwnInfo)
	if err != nil {
		d.log.Errorf("Cannot encode ping message", err)
	}

	merged := append(msg.Header, msg.Payload...)

	d.log.Tracef("Start sending discovery info")

	for {
		c.Write(merged)
		time.Sleep(1 * time.Second)
	}
}
