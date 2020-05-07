package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"

	"github.com/JoelOtter/termloop"
	"github.com/google/uuid"
	"github.com/jauhararifin/tetris"
)

func startClient(host, name, room string) {
	idUUID, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}
	id := idUUID.String()
	log.Printf("id generated: %s\n", id)

	s, err := net.ResolveUDPAddr("udp4", host)
	if err != nil {
		panic(err)
	}
	log.Printf("address resolved: %v\n", s)

	conn, err := net.DialUDP("udp4", nil, s)
	if err != nil {
		panic(err)
	}
	log.Printf("connected: %v\n", conn)

	buff := &bytes.Buffer{}
	if err := gob.NewEncoder(buff).Encode(UserMessage{
		JoinMessage: &JoinMessage{
			ID:   id,
			Name: name,
			Room: room,
		},
		RoomMessage: nil,
	}); err != nil {
		panic(err)
	}
	if _, err := conn.Write(buff.Bytes()); err != nil {
		panic(err)
	}
	log.Printf("user join message sent\n")

	initmsg := InitGameMessage{}
	msgbuff := make([]byte, 1024*1024, 1024*1024)
	if _, _, err := conn.ReadFromUDP(msgbuff); err != nil {
		panic(err)
	}
	if err := gob.NewDecoder(bytes.NewReader(msgbuff)).Decode(&initmsg); err != nil {
		panic(err)
	}
	log.Printf("init game message received: %v\n", initmsg)

	player1ID := id
	player2ID := ""
	for pid := range initmsg.Seed {
		if pid != player1ID {
			player2ID = pid
		}
	}
	log.Printf("player1ID=%s player2ID=%s\n", player1ID, player2ID)

	boardEntity1 := NewBoardPlayer(0, 2, initmsg.Width, initmsg.Height, initmsg.Seed[player1ID], func(action tetris.Action) {
		buff := &bytes.Buffer{}
		actMsg := ActionMessage{Action:action}
		if err := gob.NewEncoder(buff).Encode(actMsg); err != nil {
			log.Printf("cannot encode actio message: %v\n", err)
			return
		}

		userMsg := UserMessage{
			JoinMessage: nil,
			RoomMessage: &RoomMessage{
				ID:      id,
				Message: buff.Bytes(),
			},
		}
		buff = &bytes.Buffer{}
		if err := gob.NewEncoder(buff).Encode(userMsg); err != nil {
			log.Printf("cannot encode user message: %v\n", err)
			return
		}

		if _, err := conn.Write(buff.Bytes()); err != nil {
			log.Printf("cannot send user message: %v\n", err)
		}
	})
	boardEntity2 := NewBoardPlayer(initmsg.Width + 15, 2, initmsg.Width, initmsg.Height, initmsg.Seed[player2ID], nil)

	go func() {
		for {
			buff := make([]byte, 1024*1024, 1024*1024)
			if _, _, err := conn.ReadFromUDP(buff); err != nil {
				log.Printf("cannot read from udp: %v\n", err)
			}

			g := GameStateUpdateMessage{}
			if err := gob.NewDecoder(bytes.NewReader(buff)).Decode(&g); err != nil {
				log.Printf("cannot decode game state update message: %v\n", err)
			}

			boardEntity1.SetState(g.State[player1ID])
			boardEntity2.SetState(g.State[player2ID])
		}
	}()

	game := termloop.NewGame()
	level := termloop.NewBaseLevel(termloop.Cell{})
	level.AddEntity(boardEntity1)
	level.AddEntity(boardEntity2)
	game.Screen().SetLevel(level)
	game.Start()
}

type ActionSender func(action tetris.Action)

type boardPlayer struct {
	board               *tetris.Board
	x, y, width, height int
	score               int

	scoreText    *termloop.Text
	actionSender ActionSender
}

func NewBoardPlayer(x, y, width, height int, seed int64, actionSender ActionSender) *boardPlayer {
	b := &boardPlayer{
		width:  width,
		height: height,
		x:      x,
		y:      y,
		score:  0,

		scoreText:    termloop.NewText(x+width+3, y+8, "0", termloop.ColorWhite, termloop.ColorDefault),
		actionSender: actionSender,
	}

	b.board = tetris.NewBoard(
		tetris.WithSize(width, height),
		tetris.WithCompleteHandler(tetris.CompleteHandlerFunc(func(rows int) {
			b.score += rows * (rows + 1)
		})),
		tetris.WithGetter(tetris.NewRandomGetter(seed)),
	)

	return b
}

func (b *boardPlayer) SetState(state tetris.State) {
	b.board.SetState(state)
}

func (b *boardPlayer) Tick(ev termloop.Event) {
	if b.actionSender == nil {
		return
	}
	if ev.Type == termloop.EventKey {
		switch ev.Key {
		case termloop.KeyArrowLeft:
			b.actionSender(tetris.ActionGoLeft)
		case termloop.KeyArrowRight:
			b.actionSender(tetris.ActionGoRight)
		case termloop.KeyArrowUp:
			b.actionSender(tetris.ActionRotate)
		case termloop.KeyArrowDown:
			b.actionSender(tetris.ActionSmash)
		}
	}
}

func (b *boardPlayer) Draw(s *termloop.Screen) {
	for i := 0; i < b.width+2; i++ {
		s.RenderCell(b.x+i, b.y, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(b.x+i, b.y+b.height+1, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
	}
	for i := 0; i < b.height+2; i++ {
		s.RenderCell(b.x, b.y+i, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(b.x+b.width+1, b.y+i, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
	}

	for i := 0; i < 6; i++ {
		s.RenderCell(b.x+b.width+3+i, b.y, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(b.x+b.width+3+i, b.y+5, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(b.x+b.width+3, b.y+i, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(b.x+b.width+8, b.y+i, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
	}

	b.scoreText.SetText(fmt.Sprintf("Score: %d", b.score))
	b.scoreText.Draw(s)

	next := b.board.Next()
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			ch := rune(0)
			if next[y][x] == 1 {
				ch = '@'
			}
			s.RenderCell(b.x+b.width+4+x, b.y+1+y, &termloop.Cell{
				Fg: termloop.ColorWhite,
				Bg: termloop.ColorBlack,
				Ch: ch,
			})
		}
	}

	tiles := b.board.Render()
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			fg := termloop.ColorWhite
			bg := termloop.ColorBlack
			ch := rune(0)

			switch tiles[y][x] {
			case tetris.TileTetromino:
				fg = termloop.ColorWhite
				bg = termloop.ColorBlack
				ch = '@'
			case tetris.TileNormalBlock:
				fg = termloop.ColorWhite
				bg = termloop.ColorBlack
				ch = '#'
			case tetris.TileAdditionalBlock:
				fg = termloop.ColorRed
				bg = termloop.ColorBlack
				ch = '#'
			}

			s.RenderCell(b.x+1+x, b.y+1+y, &termloop.Cell{
				Fg: fg,
				Bg: bg,
				Ch: ch,
			})
		}
	}
}
