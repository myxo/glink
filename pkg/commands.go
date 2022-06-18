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

type AskForJoin struct {
	From         string
	To           string
	Cid          string
	Participants []string
	GroupChat    bool
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

// --------- Internal events --------------------
type ChatUpdate struct {
	Cid     string
	NewUids []string
}

func GetTypeId(cmd any) (uint16, error) {
	name := reflect.TypeOf(cmd).Name()
	switch name {
	case "NodeAnnounce":
		return 1, nil
	case "AskForJoin":
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
