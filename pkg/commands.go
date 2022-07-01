package glink

import (
	"fmt"
	"reflect"
)

type NodeAnnounce struct {
	Cid      string
	Name     string
	Endpoint string
}

type InviteForJoin struct {
	From string
	To   string
	Chat ChatInfo
}

type JoinChat struct {
	From string
	To   string
	Cid  string
}

type ChatMessage struct {
	FromUid  string
	FromName string
	ToCid    string
	Text     string
	Index    uint32
}

type ConnectInfo struct {
	MyUid  string
	MyName string
}

// -------------- Common ------------------------
type ChatInfo struct {
	Cid          string
	Participants []string
	Name         string
	Group        bool
}

// --------- Internal events --------------------
type ChatUpdate struct {
	Info    *ChatInfo
	NewUids []string
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
	}
	return 0, fmt.Errorf("Unknown command type %s", name)

}
