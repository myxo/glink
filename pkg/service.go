package glink

import (
	"bufio"
	"fmt"
	"go.uber.org/atomic"
	"os"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/juju/loggo"
)

type GlinkService struct {
	discovery      *Discovery
	Server         *Server
	Db             *Db
	OwnChatInfo    UserLightInfo
	UxEvents       chan interface{}
	log            *loggo.Logger
	conn_candidate map[string]DiscoveryInfo
	curr_msg_index map[string]atomic.Uint32
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

	own_info := db.own_info
	if own_info.Uid == "" {
		err = db.SetOwnCid(uuid.New().String())
		if err != nil {
			log.Errorf("%w", err)
		}
		own_info.Uid = db.GetOwnInfo().Uid
	}
	if own_info.Name == "" {
		own_info.Name = readName()
		db.SetOwnName(own_info.Name)
	}

	server, err := NewServer(own_info, log)
	if err != nil {
		log.Errorf("%w", err)
	}
	own_announce := NodeAnnounce{Cid: own_info.Uid, Name: own_info.Name, Endpoint: server.ListenerAddress()}

	log.Infof("Mine info. %s(%s): %s", own_announce.Name, own_announce.Cid, own_announce.Endpoint)

	go server.AcceptLoop()
	discovery := NewDiscovery(own_announce, log)
	discovery.Run()

	out := &GlinkService{discovery: discovery,
		Server:         server,
		Db:             db,
		OwnChatInfo:    own_info,
		UxEvents:       make(chan interface{}, 2),
		log:            log,
		conn_candidate: make(map[string]DiscoveryInfo),
		curr_msg_index: make(map[string]atomic.Uint32),
	}
	return out
}

func (*GlinkService) Stop() {
	//
}

func (g *GlinkService) Launch() {
	go g.serve()
}

func (g *GlinkService) UserMessage(msg ChatMessage) error {
	if msg.Text[0] == '!' {
		g.processUserCommand(msg.Text[1:])
		return nil
	}
	g.log.Tracef("Send msg to cid %s", msg.ToCid)
	msg.FromUid = g.OwnChatInfo.Uid
	msg.FromName = g.Db.own_info.Name

	index, ok := g.curr_msg_index[msg.ToCid]
	if !ok {
		index_tmp, err := g.Db.GetLastIndex(msg.ToCid)
		g.log.Debugf("Get msg index from db: %d", index_tmp)
		if err != nil {
			//g.log.Warningf("Cannot get last index: %s", err)
			// TODO: customize error?
			index_tmp = 0
		}
		index.Store(index_tmp)
		g.curr_msg_index[msg.ToCid] = index
	}
	index.Add(1)
	msg.Index = uint32(index.Load())

	g.curr_msg_index[msg.ToCid] = index

	err := g.Db.SaveMessage(msg)
	if err != nil {
		g.log.Warningf("Cannot save messages: %s", err)
	}

	err = SendToAll(g.Server, msg)
	if err != nil {
		g.log.Warningf("cannot send to all: %s", err)
		return err
	}
	g.UxEvents <- msg
	return nil
}

func (g *GlinkService) GetMessages(to_cid string) ([]ChatMessage, error) {
	msgs, err := g.Db.GetMessages(to_cid, 0, 10000000)
	if err != nil {
		return nil, err
	}
	for i := range msgs {
		// TODO: CACHE NAMES!
		msgs[i].FromName, err = g.GetNameByCid(msgs[i].FromUid)
		if err != nil {
			return nil, err
		}
	}
	return msgs, nil
}

func (g *GlinkService) GetNameByCid(cid string) (string, error) {
	return g.Db.GetNameByCid(cid)
}

func (g *GlinkService) serve() {
	g.log.Tracef("Service started")

	for {
		select {
		case new_node := <-g.discovery.NewNodes:
			g.processDiscoveryEvent(new_node)

		case ev := <-g.Server.NewEvent:
			g.processEvent(ev)
		}

	}
}

func (g *GlinkService) processEvent(ev interface{}) {
	//g.log.Tracef("Input message of type %s", reflect.TypeOf(ev).Name())
	switch ev := ev.(type) {

	case ChatMessage:
		var index atomic.Uint32
		index.Store(ev.Index)
		g.curr_msg_index[ev.ToCid] = index
		err := g.Db.SaveMessage(ev)
		if err != nil {
			g.log.Warningf("Cannot save incoming message: %s", err)
		}
		g.UxEvents <- ev

	case AskForJoin:
		g.log.Infof("Get AskForJoin msg from %s", ev.From)
		send := JoinChat{From: g.OwnChatInfo.Uid, To: ev.From, Cid: ev.Cid}
		err := g.Db.SaveNewChat(ev.Cid, ev.Participants)
		if err != nil {
			g.log.Errorf("Cannot save new chat: %s", err)
			return
		}
		g.UxEvents <- ChatUpdate{Cid: send.Cid, NewUids: []string{send.From}}
		SendToAll(g.Server, send)

	case JoinChat:
		err := g.Db.AddParticipantToChat(ev.Cid, ev.From)
		if err != nil {
			g.log.Errorf("Cannot save new chat: %s", err)

		}
		g.UxEvents <- ChatUpdate{Cid: ev.Cid, NewUids: []string{ev.From}}

	default:
		g.log.Warningf("Unknow event %s", reflect.TypeOf(ev).Name())
	}
}

func (g *GlinkService) processUserCommand(cmd string) {
	g.log.Debugf("process user command: %s", cmd)
	if strings.HasPrefix(cmd, "conn ") {
		conn_name := cmd[5:]
		node, ok := g.conn_candidate[conn_name]
		if !ok {
			g.log.Errorf("Cannot find connection named %s", conn_name)
			return
		}
		g.Db.SaveNewUid(node.ClientId, node.ClientName, node.Endpoint)
		g.Server.MakeNewConnectionTo(node.ClientId, node.Endpoint)

		cid := uuid.New().String()
		participants := []string{g.OwnChatInfo.Uid}
		msg := AskForJoin{From: g.OwnChatInfo.Uid, To: node.ClientId, Cid: cid, Participants: participants, GroupChat: false}
		err := g.Db.SaveNewChat(cid, participants)
		if err != nil {
			g.log.Errorf("Cannot save new chat: %s", err)
			return
		}
		g.UxEvents <- ChatUpdate{Cid: msg.Cid, NewUids: []string{msg.From}}
		g.log.Debugf("Sending AskForJoin")
		err = SendTo(g.Server, node.ClientId, msg)
		if err != nil {
			g.log.Errorf("Cannot send AskForJoin message: %w", err)
			return
		}
	}
}

func (g *GlinkService) processDiscoveryEvent(new_node DiscoveryInfo) {
	if new_node.ClientId == g.OwnChatInfo.Uid {
		return
	}
	if g.Db.IsKnownUid(new_node.ClientId) {
		g.log.Infof("connect to known id: %s", new_node.ClientName)
		g.Server.MakeNewConnectionTo(new_node.ClientId, new_node.Endpoint)
	} else {
		g.log.Infof("New node: %s(%s): %s", new_node.ClientName, new_node.ClientId, new_node.Endpoint)
		// TODO: save endpoints?
		// TODO: now logic of SaveNewUid is spread in several place. Need to figure out way to fix it
		g.Db.SaveNewUid(new_node.ClientId, new_node.ClientName, "")
		g.conn_candidate[new_node.ClientName] = new_node
	}
}
