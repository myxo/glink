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
	who     uint64
	chat_id uint64
}

type JoinResponce struct {
	chat_id uint64
	to      uint64
	result  bool
}

type ChatMessage struct {
	FromCid  string
	FromName string
	ToCid    string
	Payload  string
}

func GetTypeId(cmd any) (uint8, error) {
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
	}
	return 0, fmt.Errorf("Unknown command type %s", name)

}
