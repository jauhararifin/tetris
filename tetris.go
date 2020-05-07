package tetris

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Action int

const (
	ActionTick Action = iota
	ActionGoLeft
	ActionGoRight
	ActionRotate
	ActionSmash
	ActionFill
)

type Tetromino [4][4]int

var (
	TetrominoT = Tetromino{{0, 0, 0, 0}, {1, 1, 1, 0}, {0, 1, 0, 0}, {0, 0, 0, 0}}
	TetrominoL = Tetromino{{0, 1, 0, 0}, {0, 1, 0, 0}, {0, 1, 1, 0}, {0, 0, 0, 0}}
	TetrominoZ = Tetromino{{0, 1, 0, 0}, {0, 1, 1, 0}, {0, 0, 1, 0}, {0, 0, 0, 0}}
	TetrominoO = Tetromino{{0, 0, 0, 0}, {0, 1, 1, 0}, {0, 1, 1, 0}, {0, 0, 0, 0}}
	TetrominoI = Tetromino{{0, 1, 0, 0}, {0, 1, 0, 0}, {0, 1, 0, 0}, {0, 1, 0, 0}}
)

type Tile int

const (
	TileEmpty           = 0
	TileNormalBlock     = 1
	TileAdditionalBlock = 2
	TileTetromino       = 3
)

type TetrominoGetter interface {
	Next() Tetromino
}

type CompleteHandler interface {
	OnCompleted(rows int)
}

type CompleteHandlerFunc func(rows int)

func (f CompleteHandlerFunc) OnCompleted(rows int) {
	f(rows)
}

type State struct {
	Tiles              [][]Tile
	Current, Next      Tetromino
	CurrentX, CurrentY int
	IsOver             bool
}

type Board struct {
	tetrominoGetter TetrominoGetter
	completeHandler CompleteHandler
	width, height   int

	tiles              [][]Tile
	current, next      Tetromino
	currentX, currentY int
	isOver             bool

	renderFrame [][]Tile
	m           *sync.RWMutex
}

type BoardOption func(*Board)

func WithSize(width, height int) BoardOption {
	if width < 10 || height < 10 {
		panic(fmt.Errorf("minimal width x height is 10x10"))
	}
	return func(board *Board) {
		board.width = width
		board.height = height
	}
}

func WithGetter(tetrominoGetter TetrominoGetter) BoardOption {
	return func(board *Board) {
		board.tetrominoGetter = tetrominoGetter
	}
}

func WithCompleteHandler(handler CompleteHandler) BoardOption {
	return func(board *Board) {
		board.completeHandler = handler
	}
}

func NewBoard(options ...BoardOption) *Board {
	board := &Board{
		tetrominoGetter: NewRandomGetter(time.Now().UnixNano()),
		width:           10,
		height:          24,
		m:               &sync.RWMutex{},
	}
	for _, opt := range options {
		opt(board)
	}

	board.current = board.tetrominoGetter.Next()
	board.currentX = board.width/2 - 1
	board.currentY = 0
	board.next = board.tetrominoGetter.Next()
	board.isOver = false

	board.tiles = make([][]Tile, board.height, board.height)
	board.renderFrame = make([][]Tile, board.height, board.height)
	for i := 0; i < board.height; i++ {
		board.tiles[i] = make([]Tile, board.width, board.width)
		board.renderFrame[i] = make([]Tile, board.width, board.width)
		for j := 0; j < board.width; j++ {
			board.tiles[i][j] = TileEmpty
		}
	}

	return board
}

func (b *Board) SetState(state State) {
	b.m.Lock()
	b.m.Unlock()

	b.current = state.Current
	b.currentX = state.CurrentX
	b.currentY = state.CurrentY
	b.tiles = state.Tiles
	b.isOver = state.IsOver
	b.next = state.Next
}

func (b *Board) GetState() State {
	b.m.RLock()
	b.m.RUnlock()

	return State{
		Tiles:    b.tiles,
		Current:  b.current,
		Next:     b.next,
		CurrentX: b.currentX,
		CurrentY: b.currentY,
		IsOver:   b.isOver,
	}
}

func (b *Board) Next() Tetromino {
	b.m.RLock()
	b.m.RUnlock()
	return b.next
}

func (b *Board) Apply(action Action) {
	b.m.Lock()
	b.m.Unlock()

	if b.isOver {
		return
	}

	switch action {
	case ActionTick:
		b.applyTick()
	case ActionGoLeft:
		b.applyGoLeft()
	case ActionGoRight:
		b.applyGoRight()
	case ActionRotate:
		b.applyRotate()
	case ActionSmash:
		b.applySmash()
	case ActionFill:
		b.applyFill()
	}
}

func (b *Board) applyTick() {
	if b.isTouchGround() {
		b.fillTilesWithCurrentTetromino()
		b.popCompletedRows()
		b.setupNextTetromino()
		if b.isOverlapGround() {
			b.isOver = true
		}
	} else {
		b.stepDown()
	}
}

func (b *Board) isTouchGround() bool {
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if b.isTetrominoUnitOverlapBlock(y, x) || b.isTetrominoUnitTouchGround(y, x) {
				return true
			}
		}
	}
	return false
}

func (b *Board) isTetrominoUnitTouchGround(y, x int) bool {
	t := b.current
	if t[y][x] != 1 {
		return false
	}
	if b.currentY+y+1 >= b.height {
		return true
	}
	if b.currentX+x < 0 || b.currentX+x >= b.width {
		return false
	}
	if b.tiles[b.currentY+y+1][b.currentX+x] != TileEmpty {
		return true
	}
	return false
}

func (b *Board) isTetrominoUnitOverlapBlock(y, x int) bool {
	t := b.current
	if t[y][x] != 1 {
		return false
	}
	if b.currentY+y >= b.height || b.currentY+y < 0 || b.currentX+x >= b.width || b.currentX+x < 0 {
		return true
	}
	if b.tiles[b.currentY+y][b.currentX+x] != TileEmpty {
		return true
	}
	return false
}

func (b *Board) isOverlapGround() bool {
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if b.isTetrominoUnitOverlapBlock(y, x) {
				return true
			}
		}
	}
	return false
}

func (b *Board) fillTilesWithCurrentTetromino() {
	t := b.current
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if t[y][x] == 1 {
				b.tiles[b.currentY+y][b.currentX+x] = TileNormalBlock
			}
		}
	}
}

func (b *Board) popCompletedRows() {
	completedRowsCount := 0
	for y := b.height - 1; y >= 0; y-- {
		completedRowsBelow := completedRowsCount
		if b.isRowCompleted(y) {
			completedRowsCount++
		}

		if completedRowsBelow > 0 {
			for x := 0; x < b.width; x++ {
				b.tiles[y+completedRowsBelow][x] = b.tiles[y][x]
				b.tiles[y][x] = TileEmpty
			}
		}
	}

	if b.completeHandler != nil {
		b.completeHandler.OnCompleted(completedRowsCount)
	}
}

func (b *Board) isRowCompleted(row int) bool {
	for x := 0; x < b.width; x++ {
		if b.tiles[row][x] != TileNormalBlock {
			return false
		}
	}
	return true
}

func (b *Board) setupNextTetromino() {
	b.current = b.next
	b.next = b.tetrominoGetter.Next()
	b.currentY = 0
	b.currentX = b.width/2 - 1
}

func (b *Board) stepDown() {
	b.currentY++
}

func (b *Board) applyGoLeft() {
	if isHitLeftWall, _ := b.isHitWallOrTile(); isHitLeftWall {
		return
	}
	b.currentX--
}

func (b *Board) isHitWallOrTile() (hitLeftWall, hitRightWall bool) {
	t := b.current
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if t[y][x] != 1 {
				continue
			}
			hitLeft, hitRight := b.isTetrominoUnitHitWallOrTile(y, x)
			if hitLeft {
				hitLeftWall = true
			}
			if hitRight {
				hitRightWall = true
			}

			if hitLeftWall && hitRightWall {
				return
			}
		}
	}
	return
}

func (b *Board) isTetrominoUnitHitWallOrTile(y, x int) (hitLeftWall, hitRightWall bool) {
	t := b.current
	if t[y][x] != 1 {
		return false, false
	}

	if b.currentX+x <= 0 {
		hitLeftWall = true
	} else if b.currentY+y < b.height && b.tiles[b.currentY+y][b.currentX+x-1] != TileEmpty {
		hitLeftWall = true
	}

	if b.currentX+x >= b.width-1 {
		hitRightWall = true
	} else if b.currentY+y < b.height && b.tiles[b.currentY+y][b.currentX+x+1] != TileEmpty {
		hitRightWall = true
	}

	return
}

func (b *Board) applyGoRight() {
	if _, isHitRightWall := b.isHitWallOrTile(); isHitRightWall {
		return
	}
	b.currentX++
}

func (b *Board) applyRotate() {
	initialTetromino := b.current
	b.current = rotateTetromino(b.current)
	if b.isOverlapGround() {
		b.current = initialTetromino
	}
}

func rotateTetromino(t Tetromino) Tetromino {
	return Tetromino{
		{t[0][3], t[1][3], t[2][3], t[3][3],},
		{t[0][2], t[1][2], t[2][2], t[3][2],},
		{t[0][1], t[1][1], t[2][1], t[3][1],},
		{t[0][0], t[1][0], t[2][0], t[3][0],},
	}
}

func (b *Board) applySmash() {
	stepToGround := b.height

	currentTetrominoBottomY := [4]int{0, 0, 0, 0}
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			if b.current[y][x] == 1 {
				currentTetrominoBottomY[x] = b.currentY + y
			}
		}
	}

	groundY := [4]int{b.height, b.height, b.height, b.height}
	for x := 0; x < 4; x++ {
		if b.currentX+x >= b.width || b.currentX+x < 0 {
			groundY[x] = -1
			continue
		}

		for y := currentTetrominoBottomY[x]; y < b.height; y++ {
			if b.tiles[y][b.currentX+x] != TileEmpty {
				groundY[x] = y
				break
			}
		}
	}

	for x := 0; x < 4; x++ {
		for y := 3; y >= 0; y-- {
			if b.current[y][x] == 1 {
				stepToGround = minInt(stepToGround, groundY[x]-(b.currentY+y)-1)
				break
			}
		}
	}

	b.currentY += stepToGround
	b.fillTilesWithCurrentTetromino()
	b.popCompletedRows()
	b.setupNextTetromino()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (b *Board) applyFill() {
	if b.isTouchGround() {
		b.fillTilesWithCurrentTetromino()
		b.raiseGround()
		b.popCompletedRows()
		b.setupNextTetromino()
		if b.isOverlapGround() {
			b.isOver = true
		}
	} else {
		b.raiseGround()
	}
}

func (b *Board) raiseGround() {
	for y := 0; y < b.height-1; y++ {
		for x := 0; x < b.width; x++ {
			b.tiles[y][x] = b.tiles[y+1][x]
		}
	}
	for x := 0; x < b.width; x++ {
		b.tiles[b.height-1][x] = TileAdditionalBlock
	}
}

func (b *Board) Render() [][]Tile {
	b.m.RLock()
	b.m.RUnlock()

	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			b.renderFrame[y][x] = b.tiles[y][x]
		}
	}

	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			frameX := b.currentX + x
			frameY := b.currentY + y
			if b.current[y][x] == 1 && frameX >= 0 && frameX < b.width && frameY >= 0 && frameY < b.height {
				b.renderFrame[frameY][frameX] = TileTetromino
			}
		}
	}

	return b.renderFrame
}

type RandomGetter struct {
	randomizer *rand.Rand
	tetrominos []Tetromino
}

func NewRandomGetter(seed int64) *RandomGetter {
	return &RandomGetter{
		randomizer: rand.New(rand.NewSource(seed)),
		tetrominos: []Tetromino{
			TetrominoT,
			TetrominoL,
			TetrominoZ,
			TetrominoO,
			TetrominoI,
		},
	}
}

func (r *RandomGetter) Next() Tetromino {
	return r.tetrominos[r.randomizer.Int()%len(r.tetrominos)]
}

type QueueTetrominoGetter struct {
	queue []Tetromino
}

func NewQueueGetter() *QueueTetrominoGetter {
	return &QueueTetrominoGetter{queue: make([]Tetromino, 0, 0)}
}

func (q *QueueTetrominoGetter) Next() Tetromino {
	t := q.queue[0]
	q.queue = q.queue[1:]
	return t
}

func (q *QueueTetrominoGetter) Push(t ...Tetromino) {
	q.queue = append(q.queue, t...)
}
