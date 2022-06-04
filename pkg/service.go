package glink

import (
	"bufio"
	"fmt"
	"os"

	"github.com/juju/loggo"
)

type GlinkService struct {
	discovery *Discovery
	Server    *Server
	Db        *Db
	OwnCid    string
	NewMsg    chan ChatMessage
	log       *loggo.Logger
}

func readName() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name: ")
	text, _ := reader.ReadString('\n')
	return text[:len(text)-1]
}

func NewGlinkService(log *loggo.Logger, db_path string) *GlinkService {
	db, err := NewDb(db_path)
	if err != nil {
		log.Errorf("%w", err)
	}

	server, err := NewServer(log)
	if err != nil {
		log.Errorf("%w", err)
	}
	own_cid := db.own_info.Cid
	own_name := db.own_info.Name
	if own_name == "" {
		own_name = readName()
		db.SetOwnName(own_name)
	}
	own_info := NodeAnnounce{Cid: own_cid, Name: own_name, Endpoint: server.ListenerAddress()}

	log.Infof("Mine info. Cid: %s, name: %s, endpoint: %s", own_info.Cid, own_info.Name, own_info.Endpoint)

	go server.AcceptLoop()
	discovery := NewDiscovery(own_info, log)
	discovery.Run()

	return &GlinkService{discovery: discovery, Server: server, Db: db, OwnCid: own_cid, NewMsg: make(chan ChatMessage, 2), log: log}
}

func (*GlinkService) Stop() {
	//
}

func (g *GlinkService) Launch() {
	go func() {
		g.log.Tracef("Service started")

		for {
			select {
			case new_node := <-g.discovery.NewNodes:
				if new_node.ClientId == g.OwnCid {
					continue
				}
				g.log.Infof("New node: ", new_node)
				g.Server.MakeNewConnectionTo(new_node.Endpoint)
			case new_msg := <-g.Server.NewEvent:
				g.NewMsg <- new_msg
			}

		}
	}()
}

func (g *GlinkService) SendMessage(msg ChatMessage) error {
	msg.FromCid = g.OwnCid
	msg.FromName = g.Db.own_info.Name

	bytes, err := EncodeMsg(msg)
	if err != nil {
		println(err)
		return err
	}
	g.Server.SendToAll(bytes)
	g.NewMsg <- msg
	return nil
}

func (g *GlinkService) GetNameByCid(cid string) (string, error) {
	return g.Db.GetNameByCid(cid)
}
