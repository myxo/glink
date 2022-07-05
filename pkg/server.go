package glink

import (
	"errors"
	"fmt"
	"net"

	"github.com/juju/loggo"
)

func SendToAll[T any](s *Server, msg T) error {
	bytes, err := EncodeMsg(msg)
	if err != nil {
		return err
	}
	for _, conn := range s.connections {
		conn.Write(bytes.Header)
		conn.Write(bytes.Payload)
	}
	return nil
}

func SendTo[T any](s *Server, uid string, msg T) error {
	bytes, err := EncodeMsg(msg)
	if err != nil {
		return err
	}
	conn, ok := s.connections[uid]
	if !ok {
		return errors.New("Cannot get connection to " + uid)
	}
	conn.Write(bytes.Header)
	conn.Write(bytes.Payload)
	return nil
}

// TODO: mutex
type Server struct {
	listener    net.Listener
	connections map[string]net.Conn
	NewEvent    chan interface{}
	log         *loggo.Logger
	own_info    UserLightInfo
}

func NewServer(own_info UserLightInfo, log *loggo.Logger) (*Server, error) {
	listener, err := net.Listen("tcp", "localhost:0000")
	if err != nil {
		return nil, fmt.Errorf("Cannot bind: %w", err)
	}
	server := Server{
		listener:    listener,
		connections: make(map[string]net.Conn),
		NewEvent:    make(chan interface{}),
		log:         log,
		own_info:    own_info,
	}
	//go server.acceptLoop()

	return &server, nil
}

func (s *Server) ListenerAddress() string {
	return s.listener.Addr().String()
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) MakeNewConnectionTo(uid, endpoint string) error {
	c, err := net.Dial("tcp", endpoint)
	if err != nil {
		s.log.Warningf("%w", err)
		return err
	}

	s.log.Debugf("Connected to %s", c.RemoteAddr().String())

	s.connections[uid] = c
	go s.handleUserConnectoin(c, s.NewEvent)

	conn_info := ConnectInfo{MyUid: s.own_info.Uid, MyName: s.own_info.Name}
	err = SendTo(s, uid, conn_info)
	if err != nil {
		s.log.Warningf("Cannot send ConnectInfo msg: %s", err)
		return err
	}
	return nil
}

func (s *Server) AcceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.log.Warningf("Cannot accept connection: ", err)
			continue
		}
		s.log.Debugf("Accept connection from %s", conn.RemoteAddr().String())

		_, msg, err := s.readMessage(conn)
		if err != nil {
			s.log.Errorf("Failed to read message: %s, abort", err)
			conn.Close()
			return
		}
		conn_info, err := DecodeMsg[ConnectInfo](msg.Payload)
		if err != nil {
			s.log.Errorf("Failed to accept ConnectInfo message: %s, abort", err)
			conn.Close()
			return
		}

		s.log.Debugf("Get ConnectInfo msg from %s", conn_info.MyName)

		s.connections[conn_info.MyUid] = conn
		go s.handleUserConnectoin(conn, s.NewEvent)
	}
}

func (s *Server) handleUserConnectoin(c net.Conn, newEvent chan interface{}) {
	for {
		hdr, msg, err := s.readMessage(c)
		if err != nil {
			s.log.Errorf("%s", err)
			return
		}
		s.log.Tracef("Got message of type %d", hdr.MsgType)
		var ev interface{}

		switch hdr.MsgType {
		case 1:
			ev, err = DecodeMsg[NodeAnnounce](msg.Payload)
		case 2:
			ev, err = DecodeMsg[InviteForJoin](msg.Payload)
			if err == nil && ev == nil {
				s.log.Warningf("Ev is NIL!!!!!!!!")
				return
			}
		case 3:
			ev, err = DecodeMsg[JoinChat](msg.Payload)
		case 4:
			ev, err = DecodeMsg[ChatMessage](msg.Payload)
		case 5:
			ev, err = DecodeMsg[ConnectInfo](msg.Payload)
		case 6:
			ev, err = DecodeMsg[WatchedCids](msg.Payload)
		case 7:
			ev, err = DecodeMsg[HaveCidInfo](msg.Payload)
		case 8:
			ev, err = DecodeMsg[MessagesRequest](msg.Payload)
		case 9:
			ev, err = DecodeMsg[ChatMessagePack](msg.Payload)
		}
		if err != nil {
			s.log.Warningf("Cannot decode message of type %d, bytes: %s", hdr.MsgType, msg.Payload)
			return
		}
		if ev == nil {
			s.log.Warningf("Decode return no error, but ev is nil")
			return
		}
		newEvent <- ev
	}
}

func (s *Server) readMessage(c net.Conn) (MsgHeader, *MsgBytes, error) {
	msg := NewMsgBytes()
	n, err := c.Read(msg.Header)
	if err != nil {
		return MsgHeader{}, nil, err
	}
	if n < 6 {
		return MsgHeader{}, msg, errors.New("Not enought header!!!")
	}
	hdr, err := DecodeHeader(msg.Header)
	msg.Payload = make([]byte, hdr.PayloadSize)
	n, err = c.Read(msg.Payload)
	if err != nil {
		return hdr, msg, err
	}
	if n != int(hdr.PayloadSize) {
		s.log.Errorf("Not enought payload!!!")
		return hdr, msg, err
	}

	return hdr, msg, nil
}
