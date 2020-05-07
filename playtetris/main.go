package main

import (
	"fmt"
	"time"

	"github.com/JoelOtter/termloop"
	"github.com/jauhararifin/tetris"
)

func main() {
	game := termloop.NewGame()
	level := termloop.NewBaseLevel(termloop.Cell{})
	boardEntity := NewBoardPlayer(0, 0)
	level.AddEntity(boardEntity)
	game.Screen().SetLevel(level)
	game.Start()
}

type boardPlayer struct {
	board               *tetris.Board
	x, y, width, height int
	score               int

	scoreText *termloop.Text
}

func NewBoardPlayer(x, y int) *boardPlayer {
	b := &boardPlayer{
		width:  10,
		height: 24,
		x:      x,
		y:      y,
		score:  0,

		scoreText: termloop.NewText(x+10+3, y+8, "0", termloop.ColorWhite, termloop.ColorDefault),
	}

	b.board = tetris.NewBoard(
		tetris.WithSize(10, 24),
		tetris.WithCompleteHandler(tetris.CompleteHandlerFunc(func(rows int) {
			b.score += rows * (rows + 1)
		})),
	)

	ticker := time.NewTicker(500 * time.Millisecond)
	go func() {
		for range ticker.C {
			b.board.Apply(tetris.ActionTick)
		}
	}()

	return b
}

func (b *boardPlayer) Tick(ev termloop.Event) {
	if ev.Type == termloop.EventKey {
		switch ev.Key {
		case termloop.KeyArrowLeft:
			b.board.Apply(tetris.ActionGoLeft)
		case termloop.KeyArrowRight:
			b.board.Apply(tetris.ActionGoRight)
		case termloop.KeyArrowUp:
			b.board.Apply(tetris.ActionRotate)
		case termloop.KeyArrowDown:
			b.board.Apply(tetris.ActionSmash)
		case termloop.KeySpace:
			b.board.Apply(tetris.ActionFill)
		}
	}
}

func (b *boardPlayer) Draw(s *termloop.Screen) {
	for i := 0; i < b.width+2; i++ {
		s.RenderCell(i, b.y, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(i, b.y+b.height+1, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
	}
	for i := 0; i < b.height+2; i++ {
		s.RenderCell(b.x, i, &termloop.Cell{
			Fg: termloop.ColorWhite,
			Bg: termloop.ColorBlack,
			Ch: '+',
		})
		s.RenderCell(b.x+b.width+1, i, &termloop.Cell{
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
