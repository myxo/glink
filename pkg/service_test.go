package glink

import (
	"fmt"
	"testing"

	"github.com/juju/loggo"
	"github.com/stretchr/testify/require"
)

type FakeServer struct {
	evChan      chan interface{}
	connections map[Uid]string
	msgs        map[Uid][]MsgBytes
}

func NewFakeServer() *FakeServer {
	return &FakeServer{
		connections: make(map[Uid]string),
		msgs:        make(map[Uid][]MsgBytes),
	}
}

func (f *FakeServer) Run(ev chan interface{}) {
	f.evChan = ev
}

func (f *FakeServer) ListenerAddress() string {
	return "0.0.0.0:1234"
}

func (f *FakeServer) SendTo(uid Uid, msg MsgBytes) error {
	_, ok := f.connections[uid]
	if !ok {
		return fmt.Errorf("No connection %s", uid)
	}
	f.msgs[uid] = append(f.msgs[uid], msg)
	return nil
}

func (f *FakeServer) SendToAll(msg MsgBytes) error {
	for u := range f.connections {
		f.msgs[u] = append(f.msgs[u], msg)
	}
	return nil
}

func (f *FakeServer) MakeNewConnectionTo(uid Uid, endpoint string) error {
	f.connections[uid] = endpoint
	return nil
}

type FakeDiscovery struct{}

func (d *FakeDiscovery) Run(eventChan chan DiscoveryInfo) error {
	return nil
}

func (d *FakeDiscovery) Close() {}

func TestSendMessageRecieveInServer(t *testing.T) {
	server := NewFakeServer()
	server.MakeNewConnectionTo("uid", "")
	discovery := FakeDiscovery{}
	logger := loggo.GetLogger("default")
	db, err := NewDb("")
	require.Nil(t, err)

	gs, err := createService(&logger, db, server, &discovery, UserLightInfo{Name: "name", Uid: "uid"})
	require.Nil(t, err)
	sendMsg := ChatMessage{Uid: "uid", Cid: "cid", Index: 1, Text: "sample text"}
	gs.UserMessage(sendMsg)
	expect, _ := EncodeMsg(sendMsg)
	require.Equal(t, []MsgBytes{expect}, server.msgs["uid"])
}

func TestSendMessageSavedInDb(t *testing.T) {
	server := NewFakeServer()
	server.MakeNewConnectionTo("uid", "")
	discovery := FakeDiscovery{}
	logger := loggo.GetLogger("default")
	db, err := NewDb("")
	require.Nil(t, err)

	gs, err := createService(&logger, db, server, &discovery, UserLightInfo{Name: "name", Uid: "uid"})
	require.Nil(t, err)
	sendMsg := ChatMessage{Uid: "uid", Cid: "cid", Index: 1, Text: "sample text"}
	gs.UserMessage(sendMsg)
	msgs, err := db.GetMessages("cid", 0, 1000)
	require.Nil(t, err)
	require.Equal(t, []ChatMessage{sendMsg}, msgs)
}


// user command

// Msg index increases

// Network:
// ChatMessage -> save + send to UI
// Invite for join  -> save chat info
//					-> join chat
// WatchedCid -> send HasCidInfo
// HasCidInfo -> Message Request
// MessageRequest -> MessagesPack

// Discovery event -> handshake
