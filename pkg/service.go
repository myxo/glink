package glink

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"go.uber.org/atomic"

	"github.com/google/uuid"
	"github.com/juju/loggo"
)

type GlinkService struct {
	discovery       IDiscovery
	discoveryEvents chan DiscoveryInfo
	server          IServer
	serverEvents    chan interface{}
	Db              *Db
	stop            chan bool
	OwnInfo         UserLightInfo
	UxEvents        chan interface{}
	log             *loggo.Logger
	connCandidate   map[string]DiscoveryInfo
	currMsgIndex    map[Cid]atomic.Uint32
}

func readName() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your name: ")
	text, _ := reader.ReadString('\n')
	return text[:len(text)-1]
}

func NewGlinkService(log *loggo.Logger, dbPath string) (*GlinkService, error) {
	db, err := NewDb(dbPath)
	if err != nil {
		return nil, err
	}

	ownInfo := db.own_info
	if ownInfo.Uid == "" {
		err = db.SetOwnUid(Uid(uuid.New().String()))
		if err != nil {
			return nil, err
		}
		ownInfo.Uid = db.GetOwnInfo().Uid
	}
	if ownInfo.Name == "" {
		ownInfo.Name = readName()
		db.SetOwnName(ownInfo.Name)
	}
	server, err := NewServer(ownInfo, log)
	if err != nil {
		return nil, err
	}
	own_announce := NodeAnnounce{Uid: ownInfo.Uid, Name: ownInfo.Name, Endpoint: server.ListenerAddress()}

	log.Infof("Mine info. %s(%s): %s", own_announce.Name, own_announce.Uid, own_announce.Endpoint)

	discovery := NewDiscovery(own_announce, log)

	return createService(log, db, server, discovery, ownInfo)
}

func createService(
	log *loggo.Logger,
	db *Db,
	server IServer,
	discovery IDiscovery,
	ownInfo UserLightInfo,
) (*GlinkService, error) {
	out := &GlinkService{
		discovery:       discovery,
		discoveryEvents: make(chan DiscoveryInfo),
		server:          server,
		serverEvents:    make(chan interface{}),
		stop:            make(chan bool),
		Db:              db,
		OwnInfo:         ownInfo,
		UxEvents:        make(chan interface{}, 2),
		log:             log,
		connCandidate:   make(map[string]DiscoveryInfo),
		currMsgIndex:    make(map[Cid]atomic.Uint32),
	}
	err := discovery.Run(out.discoveryEvents)
	if err != nil {
		return nil, err
	}
	server.Run(out.serverEvents)
	return out, nil
}

func (g *GlinkService) Stop() {
	g.stop <- true
}

func (g *GlinkService) Launch() {
	go g.serve()
}

func (g *GlinkService) UserMessage(msg ChatMessage) error {
	if msg.Text[0] == '!' {
		g.processCommand(msg.Text[1:])
		return nil
	}
	g.log.Tracef("Send msg to cid %s", msg.Cid)
	msg.Uid = g.OwnInfo.Uid

	index, ok := g.currMsgIndex[msg.Cid]
	if !ok {
		index_tmp, err := g.Db.GetLastIndex(msg.Cid)
		g.log.Debugf("Get msg index from db: %d", index_tmp)
		if err != nil {
			//g.log.Warningf("Cannot get last index: %s", err)
			// TODO: customize error?
			index_tmp = 0
		}
		index.Store(index_tmp)
		g.currMsgIndex[msg.Cid] = index
	}
	index.Add(1)
	msg.Index = uint32(index.Load())

	g.currMsgIndex[msg.Cid] = index

	err := g.Db.SaveMessage(msg)
	if err != nil {
		g.log.Warningf("Cannot save messages: %s", err)
	}

	err = SendToAll(g.server, msg)
	if err != nil {
		g.log.Warningf("cannot send to all: %s", err)
		return err
	}
	g.UxEvents <- msg
	return nil
}

func (g *GlinkService) GetMessages(to_cid Cid) ([]ChatMessage, error) {
	return g.Db.GetMessages(to_cid, 0, 10000000)
}

func (g *GlinkService) GetNameByCid(uid Uid) (string, error) {
	return g.Db.GetNameByUid(uid)
}

func (g *GlinkService) serve() {
	g.log.Tracef("Service started")

	for {
		select {
		case new_node := <-g.discoveryEvents:
			g.processDiscoveryEvent(new_node)

		case ev := <-g.serverEvents:
			g.processNetworkEvent(ev)

		case <-g.stop:
			return
		}

	}
}

func (g *GlinkService) processNetworkEvent(ev interface{}) {
	g.log.Debugf("Input message of type %s", reflect.TypeOf(ev).Name())
	switch ev := ev.(type) {

	case ChatMessage:
		var index atomic.Uint32
		index.Store(ev.Index)
		g.currMsgIndex[ev.Cid] = index
		err := g.Db.SaveMessage(ev)
		if err != nil {
			g.log.Warningf("Cannot save incoming message: %s", err)
		}
		g.UxEvents <- ev

	case InviteForJoin:
		g.log.Infof("Get InviteForJoin msg from %s(%s)", ev.Chat.Name, ev.From)
		send := JoinChat{From: g.OwnInfo.Uid, To: ev.From, Cid: ev.Chat.Cid}
		chatName := ev.Chat.Name
		if !ev.Chat.Group {
			username, err := g.GetNameByCid(ev.From)
			if err != nil {
				g.log.Errorf("Cannot get name by cid: %s", err)
				return
			}
			chatName = username
		}
		err := g.Db.SaveNewChat(ev.Chat.Cid, chatName, ev.Chat.Participants)
		if err != nil {
			g.log.Errorf("Cannot save new chat: %s", err)
			return
		}
		info := &ChatInfo{Cid: ev.Chat.Cid, Name: chatName, Participants: ev.Chat.Participants, Group: ev.Chat.Group}
		g.UxEvents <- ChatUpdate{Info: info, NewUids: []Uid{send.From}}
		SendToAll(g.server, send)

	case JoinChat:
		err := g.Db.AddParticipantToChat(ev.Cid, ev.From)
		if err != nil {
			g.log.Errorf("Cannot save new chat: %s", err)

		}
		info, err := g.Db.GetChatInfo(ev.Cid)
		if err != nil {
			g.log.Errorf("Cannot get chat info for cid %s", ev.Cid)
			return
		}
		g.UxEvents <- ChatUpdate{Info: info, NewUids: []Uid{ev.From}}

	case WatchedCids:
		vc, err := g.GetVectorClockOfKnownCids(ev.Cids)
		if err != nil {
			g.log.Errorf("Cannot get vector clock of cids [%v], error: %s", ev.Cids, err)
		}
		if g.log.IsTraceEnabled() {
			pp, _ := json.MarshalIndent(vc, "", "  ")
			g.log.Tracef("Return vector clock to %s\n%s", ev.From, pp)
		}
		err = SendTo(g.server, ev.From, HaveCidInfo{From: g.OwnInfo.Uid, To: ev.From, ChatsVectorClock: vc})
		if err != nil {
			g.log.Errorf("Cannot send vector clock to %s, error: %s", ev.From, err)
		}

	case HaveCidInfo:
		req, err := g.GenerateMessagesRequest(ev.ChatsVectorClock)
		if err != nil {
			g.log.Errorf("Cannot enerate message request: %s", err)
		}
		if len(req) == 0 {
			return
		}
		err = SendTo(g.server, ev.From, MessagesRequest{From: g.OwnInfo.Uid, To: ev.From, VectorClockFrom: req})
		if err != nil {
			g.log.Errorf("Cannot send message request: %s", err)
		}

	case MessagesRequest:
		msgs, err := g.Db.GetMessagesByVectorClock(ev.VectorClockFrom)
		if err != nil {
			g.log.Errorf("Cannot get messages by vector: %s", err)
			return
		}
		SendTo(g.server, ev.From, ChatMessagePack{From: g.OwnInfo.Uid, To: ev.From, Messages: msgs})

	case ChatMessagePack:
		for _, msg := range ev.Messages {
			err := g.Db.SaveMessage(msg)
			if err != nil {
				g.log.Errorf("Cannot save msg to db: %s", err)
				return
			}
		}
		g.UxEvents <- ev

	default:
		g.log.Warningf("Service.processNetworkEvent: unknown event %s", reflect.TypeOf(ev).Name())
	}
}

func (g *GlinkService) processCommand(cmd string) {
	g.log.Debugf("process user command: %s", cmd)
	if strings.HasPrefix(cmd, "conn ") {
		conn_name := cmd[5:]
		node, ok := g.connCandidate[conn_name]
		if !ok {
			g.log.Errorf("Cannot find connection named %s", conn_name)
			return
		}
		g.Db.SaveNewUid(node.ClientId, node.ClientName, node.Endpoint)
		g.initHandshake(node.ClientId, node.Endpoint)

		cid := Cid(uuid.New().String())
		participants := []Uid{g.OwnInfo.Uid}
		chatInfo := ChatInfo{Cid: cid, Participants: participants, Group: false}
		msg := InviteForJoin{From: g.OwnInfo.Uid, To: node.ClientId, Chat: chatInfo}
		err := g.Db.SaveNewChat(cid, node.ClientName, participants)
		if err != nil {
			g.log.Errorf("Cannot save new chat: %s", err)
			return
		}
		g.log.Debugf("Sending AskForJoin")
		err = SendTo(g.server, node.ClientId, msg)
		if err != nil {
			g.log.Errorf("Cannot send AskForJoin message: %s", err)
			return
		}
		chatInfo.Name = node.ClientName
		g.UxEvents <- ChatUpdate{Info: &chatInfo, NewUids: []Uid{msg.From}}
	}
}

func (g *GlinkService) processDiscoveryEvent(new_node DiscoveryInfo) {
	if new_node.ClientId == g.OwnInfo.Uid {
		return
	}
	if g.Db.IsKnownUid(new_node.ClientId) {
		g.log.Infof("connect to known id: %s", new_node.ClientName)
		g.initHandshake(new_node.ClientId, new_node.Endpoint)
	} else {
		g.log.Infof("New node: %s(%s): %s", new_node.ClientName, new_node.ClientId, new_node.Endpoint)
		// TODO: save endpoints?
		// TODO: now logic of SaveNewUid is spread in several place. Need to figure out way to fix it
		g.Db.SaveNewUid(new_node.ClientId, new_node.ClientName, "")
		g.connCandidate[new_node.ClientName] = new_node
	}
}

func (g *GlinkService) initHandshake(uid Uid, endpoint string) error {
	err := g.server.MakeNewConnectionTo(uid, endpoint)
	if err != nil {
		return err
	}
	watchedCids, err := g.GetWatchedCids()
	if err != nil {
		return err
	}
	err = SendTo(g.server, uid, WatchedCids{From: g.OwnInfo.Uid, To: uid, Cids: watchedCids})
	if err != nil {
		return err
	}
	return nil
}

func (g *GlinkService) GetWatchedCids() ([]Cid, error) {
	chats, err := g.Db.GetChats(false)
	if err != nil {
		return nil, err
	}
	cids := make([]Cid, len(chats))
	for _, chat := range chats {
		cids = append(cids, chat.Cid)
	}
	return cids, nil
}

func (g *GlinkService) GetListOfKnownCids() ([]UserLightInfo, error) {
	return g.Db.GetUsersInfo()
}

func (g *GlinkService) GetVectorClockOfKnownCids(cids []Cid) (map[Cid]VectorClock, error) {
	// TODO: caching
	return g.Db.GetVectorClockOfCids(cids)
}

func (g *GlinkService) GenerateMessagesRequest(vectorClock map[Cid]VectorClock) (map[Cid]VectorClock, error) {
	// TODO: caching
	cids := make([]Cid, 0, len(vectorClock))
	for cid := range vectorClock {
		cids = append(cids, cid)
	}
	curVector, err := g.Db.GetVectorClockOfCids(cids)
	if err != nil {
		return nil, err
	}

	res := make(map[Cid]VectorClock)

	for cid, vector := range vectorClock {
		subVec := curVector[cid]
		for uid, index := range vector {
			curIndex := subVec[uid]
			if curIndex < index {
				if res[cid] == nil {
					res[cid] = make(VectorClock)
				}
				res[cid][uid] = curIndex
			}
		}
	}
	return res, nil
}
