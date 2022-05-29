package glink

import (
	"encoding/binary"
	"encoding/json"
)

type MsgHeader struct {
	PayloadSize uint32
	MsgType     uint8
	padding     uint8
}

type MsgBytes struct {
	Header  []byte
	Payload []byte
}

func NewMsgBytes() *MsgBytes {
	return &MsgBytes{Header: make([]byte, 6, 6)}
}

func EncodeHeader(hdr MsgHeader) ([]byte, error) {
	header := make([]byte, 6)
	binary.LittleEndian.PutUint32(header, uint32(hdr.PayloadSize))
	return header, nil
}

func DecodeHeader(bytes []byte) (MsgHeader, error) {
	var res MsgHeader
	res.PayloadSize = uint32(binary.LittleEndian.Uint32(bytes))
	return res, nil
}

func EncodeMsg(msg any) (MsgBytes, error) {
	j, err := json.Marshal(msg)
	if err != nil {
		return MsgBytes{}, err
	}

	res := MsgBytes{
		Payload: j,
		Header:  make([]byte, 6),
	}

	msg_type, err := GetTypeId(msg)
	if err != nil {
		return res, err
	}

	hdr := MsgHeader{
		PayloadSize: uint32(len(j)),
		MsgType:     msg_type,
	}
	res.Header, err = EncodeHeader(hdr)

	return res, err
}

func DecodeMsg[T any](hdr MsgHeader, payload []byte) (T, error) {
	var res T

	err := json.Unmarshal(payload, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}
