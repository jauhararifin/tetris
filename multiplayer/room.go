package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jauhararifin/tetris"
)

type MessageSender interface {
	Send(playerID string, msg []byte) error
}

type MessageSenderFunc func(playerID string, msg []byte) error

func (f MessageSenderFunc) Send(playerID string, msg []byte) error {
	return f(playerID, msg)
}

type Player struct {
	ID   string
	Name string
}

type Room struct {
	m                *sync.Mutex
	randomizer       *rand.Rand
	player1, player2 Player
	board1, board2   *tetris.Board
	sender           MessageSender
	isStarted        bool
	updateTicker     *time.Ticker
	gameTicker       *time.Ticker
}

func NewRoom(sender MessageSender) *Room {
	return &Room{
		m:            &sync.Mutex{},
		randomizer:   rand.New(rand.NewSource(time.Now().UnixNano())),
		player1:      Player{},
		player2:      Player{},
		board1:       nil,
		board2:       nil,
		sender:       sender,
		isStarted:    false,
		updateTicker: nil,
		gameTicker:   nil,
	}
}

func (r *Room) OnPlayerJoin(player Player) error {
	if player.ID == "" || player.Name == "" {
		return fmt.Errorf("player id or name cannot empty")
	}

	if r.player1.ID == "" {
		r.player1 = player
		return nil
	}

	if r.player2.ID == "" {
		r.player2 = player
		r.initGame()
		return nil
	}

	return fmt.Errorf("room already full")
}

func (r *Room) OnPlayerLeave(player Player) error {
	if r.player1 == player {
		r.player1 = r.player2
		r.player2 = Player{}
		r.stopGame()
		return nil
	}

	if r.player2 == player {
		r.player2 = Player{}
		r.stopGame()
		return nil
	}

	return fmt.Errorf("no such player")
}

type InitGameMessage struct {
	Seed          map[string]int64
	FPS           int
	Width, Height int
}

func (r *Room) initGame() {
	r.m.Lock()
	defer r.m.Unlock()

	width, height := 10, 24
	fps := 24
	randA, randB := r.randomizer.Int63(), r.randomizer.Int63()
	r.board1 = tetris.NewBoard(
		tetris.WithSize(width, height),
		tetris.WithGetter(tetris.NewRandomGetter(randA)),
		tetris.WithCompleteHandler(tetris.CompleteHandlerFunc(func(rows int) {
			for i := 0; i < rows; i++ {
				r.board2.Apply(tetris.ActionFill)
			}
		})),
	)
	r.board2 = tetris.NewBoard(
		tetris.WithSize(width, height),
		tetris.WithGetter(tetris.NewRandomGetter(randB)),
		tetris.WithCompleteHandler(tetris.CompleteHandlerFunc(func(rows int) {
			for i := 0; i < rows; i++ {
				r.board1.Apply(tetris.ActionFill)
			}
		})),
	)
	r.isStarted = true

	buff := &bytes.Buffer{}
	if err := gob.NewEncoder(buff).Encode(InitGameMessage{
		Seed:   map[string]int64{r.player1.ID: randA, r.player2.ID: randB},
		FPS:    fps,
		Width:  width,
		Height: height,
	}); err != nil {
		log.Printf("cannot encode init message: %v\n", err)
	}

	if err := r.sender.Send(r.player1.ID, buff.Bytes()); err != nil {
		log.Printf("cannot send init message to player 1 (%s): %v\n", r.player1.ID, err)
	}

	if err := r.sender.Send(r.player2.ID, buff.Bytes()); err != nil {
		log.Printf("cannot send init message to player 2 (%s): %v\n", r.player2.ID, err)
	}

	go r.startGame(fps)
}

type GameStateUpdateMessage struct {
	State map[string]tetris.State
}

func (r *Room) startGame(fps int) {
	time.Sleep(3 * time.Second)

	ms := time.Duration(1000 / fps)
	r.updateTicker = time.NewTicker(ms * time.Millisecond)
	r.gameTicker = time.NewTicker(750 * time.Millisecond)

	go func() {
		for range r.gameTicker.C {
			r.board1.Apply(tetris.ActionTick)
			r.board2.Apply(tetris.ActionTick)
		}
	}()

	go func() {
		for range r.updateTicker.C {
			buff := &bytes.Buffer{}
			if err := gob.NewEncoder(buff).Encode(GameStateUpdateMessage{
				State: map[string]tetris.State{
					r.player1.ID: r.board1.GetState(),
					r.player2.ID: r.board2.GetState(),
				},
			}); err != nil {
				log.Printf("cannot encode game state update message: %v\n", err)
			}

			if err := r.sender.Send(r.player1.ID, buff.Bytes()); err != nil {
				log.Printf("cannot send game state update message to player 1 (%s): %v\n", r.player1.ID, err)
			}

			if err := r.sender.Send(r.player2.ID, buff.Bytes()); err != nil {
				log.Printf("cannot send game state update message to player 2 (%s): %v\n", r.player2.ID, err)
			}
		}
	}()
}

func (r *Room) stopGame() {
	r.m.Lock()
	defer r.m.Unlock()
	r.board1 = nil
	r.board2 = nil
	r.isStarted = false
	if r.updateTicker != nil {
		r.updateTicker.Stop()
	}
}

type ActionMessage struct {
	Action tetris.Action
}

func (r *Room) OnMessage(playerID string, msg []byte) {
	actionMsg := &ActionMessage{}
	if err := gob.NewDecoder(bytes.NewReader(msg)).Decode(&actionMsg); err != nil {
		log.Printf("cannot parse action message from playerid %s: %v\n", playerID, err)
		return
	}

	if playerID == r.player1.ID {
		r.board1.Apply(actionMsg.Action)
		return
	}

	if playerID == r.player2.ID {
		r.board2.Apply(actionMsg.Action)
		return
	}

	log.Printf("unrecognized player id: %s\n", playerID)
}
