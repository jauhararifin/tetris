package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"net"
)

type server struct {
	conn     *net.UDPConn
	rooms    map[string]*Room
	userAddr map[string]*net.UDPAddr
	userRoom map[string]string
}

func (s *server) OnUserJoin(id, name, room string, addr *net.UDPAddr) {
	r, ok := s.rooms[room]
	if !ok {
		r = NewRoom(s)
		s.rooms[room] = r
	}

	s.userAddr[id] = addr
	s.userRoom[id] = room
	r.OnPlayerJoin(Player{
		ID:   id,
		Name: name,
	})
}

func (s *server) Send(playerID string, msg []byte) error {
	addr, ok := s.userAddr[playerID]
	if !ok {
		log.Printf("cannot get user addr with id=%s\n", playerID)
	}
	_, err := s.conn.WriteToUDP(msg, addr)
	return err
}

func (s *server) OnUserMessage(id string, msg []byte) {
	rid, ok := s.userRoom[id]
	if !ok {
		log.Printf("cannot get user room with id=%s\n", id)
	}

	r, ok := s.rooms[rid]
	if !ok {
		log.Printf("cannot get room with id=%s\n", rid)
	}

	r.OnMessage(id, msg)
}

type JoinMessage struct {
	ID   string
	Name string
	Room string
}

type RoomMessage struct {
	ID      string
	Message []byte
}

type UserMessage struct {
	JoinMessage *JoinMessage
	RoomMessage *RoomMessage
}

func startServer() {
	s, err := net.ResolveUDPAddr("udp4", ":8123")
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp4", s)
	if err != nil {
		panic(err)
	}

	gameServer := &server{
		conn:     conn,
		rooms:    make(map[string]*Room),
		userAddr: make(map[string]*net.UDPAddr),
		userRoom: make(map[string]string),
	}

	for {
		buff := make([]byte, 1024*1024, 1024*1024)
		_, addr, err := conn.ReadFromUDP(buff)
		if err != nil {
			log.Printf("cannot read from udp: %v\n", err)
			continue
		}

		userMsg := UserMessage{}
		if err := gob.NewDecoder(bytes.NewReader(buff)).Decode(&userMsg); err != nil {
			log.Printf("cannot parse user message: %v\n", err)
		}
		log.Printf("got user message: %#v\n", userMsg)

		if userMsg.JoinMessage != nil {
			joinMsg := userMsg.JoinMessage
			gameServer.OnUserJoin(joinMsg.ID, joinMsg.Name, joinMsg.Room, addr)
		}

		if userMsg.RoomMessage != nil {
			roomMsg := userMsg.RoomMessage
			gameServer.OnUserMessage(roomMsg.ID, roomMsg.Message)
		}
	}
}
