package glink

import (
	"fmt"
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
	ClientId   Uid
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

func (d *Discovery) Run() error {
	err := d.serve()
	if err != nil {
		return err
	}
	go d.ping()
	return nil
}

func (*Discovery) Close() {

}

func (d *Discovery) serve() error {
	addr, err := net.ResolveUDPAddr("udp", srvAddr)
	if err != nil {
		return err
	}

	l, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("Cannot listen multicast: %s", err)
	}

	go func() {
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
				d.NewNodes <- DiscoveryInfo{ClientId: msg.Uid, ClientName: msg.Name, Endpoint: msg.Endpoint}
			}
		}
	}()
	return nil
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
