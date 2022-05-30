package glink

import (
	"fmt"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/juju/loggo"
)

var connMap = &sync.Map{}

type Server struct {
	listener    net.Listener
	connections []net.Conn
	NewEvent    chan ChatMessage
	log         *loggo.Logger
}

func NewServer(log *loggo.Logger) (*Server, error) {
	listener, err := net.Listen("tcp", "localhost:0000")
	if err != nil {
		return nil, fmt.Errorf("Cannot bind: %w", err)
	}
	server := Server{listener: listener, NewEvent: make(chan ChatMessage), log: log}
	//go server.acceptLoop()

	return &server, nil
}

func (s *Server) ListenerAddress() string {
	return s.listener.Addr().String()
}

func (s *Server) Close() {
	s.listener.Close()
}

func (s *Server) MakeNewConnectionTo(address string) error {
	c, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println(err)
		return err
	}

	s.log.Infof("Connected to %s", c.RemoteAddr().String())

	s.connections = append(s.connections, c)
	return nil
}

func (s *Server) SendToAll(msg MsgBytes) {
	for _, conn := range s.connections {
		conn.Write(msg.Header)
		conn.Write(msg.Payload)
	}
}

func (s *Server) AcceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println("Cannot accept connection: ", err)
			continue
		}
		s.log.Infof("Accept connection from %s", conn.RemoteAddr().String())

		id := uuid.New().String()
		go handleUserConnectoin(id, conn, s.NewEvent)
	}
}

func handleUserConnectoin(id string, c net.Conn, newEvent chan ChatMessage) {
	connMap.Store(id, c)
	defer func() {
		c.Close()
		connMap.Delete(id)
	}()

	for {
		msg := NewMsgBytes()
		n, err := c.Read(msg.Header)
		if err != nil {
			return
		}
		if n < 6 {
			println("Not enought header!!!")
			return
		}
		hdr, err := DecodeHeader(msg.Header)
		msg.Payload = make([]byte, hdr.PayloadSize)
		n, err = c.Read(msg.Payload)
		if err != nil {
			return
		}
		if n != int(hdr.PayloadSize) {
			println("Not enought payload!!!")
			continue
		}
		chat_msg, err := DecodeMsg[ChatMessage](hdr, msg.Payload)
		newEvent <- chat_msg
		//fmt.Printf("Message from %d: %s", chat_msg.FromId, chat_msg.Payload)

		// connMap.Range(func(key, value interface{}) bool {
		// 	if conn, ok := value.(net.Conn); ok {
		// 		conn.Write([]byte(chat_msg.Payload))
		// 	}

		// 	return true
		// })
	}
}
