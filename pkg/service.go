package glink

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/juju/loggo"
)

type GlinkService struct {
	discovery *Discovery
	Server    *Server
	OwnCid    string
	NewMsg    chan ChatMessage
	log       *loggo.Logger
}

func NewGlinkService(log *loggo.Logger) *GlinkService {
	server, err := NewServer(log)
	if err != nil {
		log.Errorf("%w", err)
	}
	own_cid := uuid.New().String()
	own_info := NodeAnnounce{Cid: own_cid, Name: "my_name", Endpoint: server.ListenerAddress()}

	log.Infof("Mine info: ", own_info)

	go server.AcceptLoop()
	discovery := NewDiscovery(own_info)
	discovery.Run()

	return &GlinkService{discovery: discovery, Server: server, OwnCid: own_cid, NewMsg: make(chan ChatMessage), log: log}
}

func (*GlinkService) Stop() {
	//
}

func (g *GlinkService) Launch() {
	go func() {
		fmt.Println("Service started")

		for {
			select {
			case new_node := <-g.discovery.NewNodes:
				fmt.Println("New node: ", new_node)
				g.Server.MakeNewConnectionTo(new_node.Endpoint)
			case new_msg := <-g.Server.NewEvent:
				g.NewMsg <- new_msg
			}

		}
	}()
}

func (g *GlinkService) SendMessage(msg ChatMessage) error {

	bytes, err := EncodeMsg(msg)
	if err != nil {
		println(err)
		return err
	}
	g.Server.SendToAll(bytes)
	return nil
}
