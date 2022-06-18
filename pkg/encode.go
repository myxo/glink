package glink

import (
	"encoding/binary"
	"encoding/json"
)

type MsgHeader struct {
	PayloadSize uint32
	MsgType     uint16 // binary packet do not have PutUint8... wtf?
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
	binary.LittleEndian.PutUint32(header, hdr.PayloadSize)
	binary.LittleEndian.PutUint16(header[4:], hdr.MsgType)
	return header, nil
}

func DecodeHeader(bytes []byte) (MsgHeader, error) {
	var res MsgHeader
	res.PayloadSize = binary.LittleEndian.Uint32(bytes)
	res.MsgType = binary.LittleEndian.Uint16(bytes[4:])
	return res, nil
}

func EncodeMsg(msg any) (MsgBytes, error) {
	j, err := json.Marshal(msg)
	if err != nil {
		return MsgBytes{}, err
	}

	res := MsgBytes{
		Payload: j,
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

func DecodeMsg[T any](payload []byte) (T, error) {
	var res T

	err := json.Unmarshal(payload, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}
