package glink

import (
	"fmt"
	"github.com/google/uuid"
	"log"
)

type GlinkService struct {
	discovery *Discovery
	Server    *Server
	OwnCid    string
}

func NewGlinkService() *GlinkService {
	server, err := NewServer()
	if err != nil {
		log.Fatalln(err)
	}
	own_cid := uuid.New().String()
	own_info := NodeAnnounce{Cid: own_cid, Name: "my_name", Endpoint: server.ListenerAddress()}

	fmt.Println("Mine info: ", own_info)

	go server.AcceptLoop()
	discovery := NewDiscovery(own_info)
	discovery.Run()

	return &GlinkService{discovery: discovery, Server: server, OwnCid: own_cid}
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
