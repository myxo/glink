package glink

import (
	"log"
	"net"
	"sync"
	"time"
)

const (
	srvAddr         = "224.0.0.1:9999"
	maxDatagramSize = 8192
)

var knownNodes = &sync.Map{}

type DiscoveryInfo struct {
	ClientId string
	Endpoint string
}

type Discovery struct {
	NewNodes chan DiscoveryInfo
	OwnInfo  NodeAnnounce
}

func NewDiscovery(own_info NodeAnnounce) *Discovery {
	return &Discovery{NewNodes: make(chan DiscoveryInfo), OwnInfo: own_info}
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
		log.Fatal(err)
	}

	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		log.Fatalln("Cannot listen multicast", err)
	}
	l.SetReadBuffer(maxDatagramSize)
	for {
		buffer := make([]byte, maxDatagramSize)
		n, src, err := l.ReadFromUDP(buffer)
		if err != nil || n == 0 {
			log.Print("ReadFromUDP failed:", err)
			continue
		}

		hdr, err := DecodeHeader(buffer)
		if err != nil {
			log.Print("Cannot decode header:", err)
			continue
		}

		payload := buffer[6 : 6+hdr.PayloadSize]
		msg, err := DecodeMsg[NodeAnnounce](hdr, payload)
		if err != nil {
			log.Print("Cannot decode payload:", payload)
			continue
		}

		if _, has := knownNodes.Load(src.String()); !has {
			knownNodes.Store(src.String(), nil)
			d.NewNodes <- DiscoveryInfo{ClientId: msg.Cid, Endpoint: msg.Endpoint}
		}
		//h(src, n, b)
	}
}

func (d *Discovery) ping() {
	addr, err := net.ResolveUDPAddr("udp", srvAddr)
	if err != nil {
		log.Fatal(err)
	}
	c, err := net.DialUDP("udp", nil, addr)
	msg, err := EncodeMsg(d.OwnInfo)
	if err != nil {
		log.Fatal("Cannot encode ping message", err)
	}

	merged := append(msg.Header, msg.Payload...)

	for {
		c.Write(merged)
		time.Sleep(1 * time.Second)
	}
}
