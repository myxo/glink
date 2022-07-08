package glink

import (
	"fmt"
	"reflect"
)

type Uid string
type Cid string

type NodeAnnounce struct {
	Uid      Uid
	Name     string
	Endpoint string
}

type InviteForJoin struct {
	From Uid
	To   Uid
	Chat ChatInfo
}

type JoinChat struct {
	From Uid
	To   Uid
	Cid  Cid
}

type ChatMessage struct {
	Uid   Uid
	Cid   Cid
	Text  string
	Index uint32
}

type ConnectInfo struct {
	MyUid  Uid
	MyName string
}

type WatchedCids struct {
	From Uid
	To   Uid
	Cids []Cid
}

type HaveCidInfo struct {
	From             Uid
	To               Uid
	ChatsVectorClock map[Cid]VectorClock
}

type MessagesRequest struct {
	From            Uid
	To              Uid
	VectorClockFrom map[Cid]VectorClock
}

type ChatMessagePack struct {
	From     Uid
	To       Uid
	Messages []ChatMessage
}

// -------------- Common ------------------------
type ChatInfo struct {
	Cid          Cid
	Participants []Uid
	Name         string
	Group        bool
}

type VectorClockElem struct {
	Uid   Uid
	Index uint32
}

type VectorClock = map[Uid]uint32

// --------- Internal events --------------------
type ChatUpdate struct {
	Info    *ChatInfo
	NewUids []Uid
}

func GetTypeId(cmd any) (uint16, error) {
	name := reflect.TypeOf(cmd).Name()
	switch name {
	case "NodeAnnounce":
		return 1, nil
	case "InviteForJoin":
		return 2, nil
	case "JoinResponce":
		return 3, nil
	case "ChatMessage":
		return 4, nil
	case "ConnectInfo":
		return 5, nil
	case "WatchedCids":
		return 6, nil
	case "HaveCidInfo":
		return 7, nil
	case "MessagesRequest":
		return 8, nil
	case "ChatMessagePack":
		return 9, nil
	}

	return 0, fmt.Errorf("Unknown command type %s", name)

}
