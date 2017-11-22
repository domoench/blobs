package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/mgutz/logxi/v1"
	"github.com/nsf/termbox-go"
)

// TODO
//  - move things into their own files

var gameLog log.Logger

func init() {
	gameLog = log.NewLogger(os.Stderr, "game")
}

const (
	nonEmptyWeight = 0.95
)

// UTILITIES
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// inMap returns true if the point is within the map's boundaries
func inMap(g *game, x, y int) bool {
	return x >= 0 && x < g.mapW && y >= 0 && y < g.mapH
}

type player struct {
	name   string
	symbol rune // Player's blob symbol
	color  termbox.Attribute
	g      *game
	x      int
	y      int
}

func (p *player) updatePos(x, y int) {
	if x >= 0 && x < p.g.mapW {
		p.x = x
	}
	if y >= 0 && y < p.g.mapH {
		p.y = y
	}
}

// Directions: indexes into 9-element adjacent player slices
const (
	UPLEFT = iota
	UP
	UPRIGHT
	LEFT
	CENTER
	RIGHT
	DOWNLEFT
	DOWN
	DOWNRIGHT
)

// adjPoints returns a slice of the 9 owners adjacent to and
// including the cell at (x,y). If adjacent points are unowned or outside
// the map boundaries, their owners are nil.
func adjacent(g *game, x, y int) []*player {
	adj := make([]*player, 9)
	i := 0
	for j := x - 1; j <= x+1; j++ {
		for k := y - 1; k <= y+1; k++ {
			if !inMap(g, j, k) {
				adj[i] = nil
			} else {
				adj[i] = g.curr[j][k]
			}
			i++
		}
	}
	return adj
}

// Game maintains game state
type game struct {
	players []*player
	input   *input
	mapW    int
	mapH    int
	curr    blobMap // current blob map
	next    blobMap // next blob map to be drawn
}

type blobMap [][]*player

func newBlobMap(w, h int) blobMap {
	bm := make([][]*player, w)
	for x := range bm {
		bm[x] = make([]*player, h)
	}
	return bm
}

// clear sets all the references in this blobMap to nil
func (bm blobMap) clear() {
	for x := range bm {
		for y := range bm[x] {
			bm[x][y] = nil
		}
	}
}

// addPlayer adds a new player (with no blobs) to the game
func (g *game) addPlayer(name string, symbol rune, color termbox.Attribute, x, y int) {
	p := &player{
		name:   name,
		symbol: symbol,
		color:  color,
		g:      g,
		x:      x,
		y:      y,
	}
	g.players = append(g.players, p)

	playerNames := make([]string, len(g.players))
	for i := range g.players {
		playerNames[i] = g.players[i].name
	}
	gameLog.Debug("player added", "player", name, "players", playerNames)
}

// newGame initializes the game's state
func newGame() *game {
	gameLog.Info("initializing new game...")

	mapWidth, mapHeight := 70, 40
	g := game{
		input: newInput(),
		mapW:  mapWidth,
		mapH:  mapHeight,
		curr:  newBlobMap(mapWidth, mapHeight),
		next:  newBlobMap(mapWidth, mapHeight),
	}
	g.addPlayer("david", 'D', termbox.ColorGreen, 0, 0)
	g.addPlayer("enemy", 'E', termbox.ColorBlue, 20, 20)

	// Start players with random blob seeds
	blobsPerPlayer := 1
	for _, pl := range g.players {
		// TODO: Random walk generate the blobs
		for i := 0; i < blobsPerPlayer; i++ {
			x := rand.Intn(g.mapW)
			y := rand.Intn(g.mapH)
			// curr and next start the same
			g.curr[x][y] = pl
			g.next[x][y] = pl
		}
	}

	return &g
}

// update calculates all the entity interactions since the last loop
// iteration and updates their states accordingly.
func (g *game) update() {

	// Determine what the next map should be based on the current one's cell interactions
	// TODO multithread this if you hit a bottleneck
	for x := range g.curr {
		for y := range g.curr[x] {
			adj := adjacent(g, x, y)

			n := next(g, adj, rand.Float64())

			g.next[x][y] = n
		}
	}
}

func adjString(adj []*player) []string {
	names := make([]string, len(adj))
	for _, p := range adj {
		if p != nil {
			names = append(names, p.name)
		} else {
			names = append(names, "nil")
		}
	}
	return names
}

// next determines what player will own the center cell in the given adjacent
// slice. x should be a randomly generated float between [0,1), which will be
// used as the dice roll.
func next(g *game, adj []*player, x float64) *player {
	// Always have 60% chance of not changing at all
	f := rand.Float64()
	if f >= 0 && f < 0.90 {
		return adj[CENTER]
	}

	// TODO wasteful to allocate these data structs on every call, they're reusable
	// To count the number of adjacent cells owned by each player.
	count := make([]float64, len(g.players))
	// Map from player references to their index
	m := make(map[*player]int)
	for i, pl := range g.players {
		m[pl] = i
	}

	// Note: currently counting the center cell just like the others
	nonEmptyCells := 0.0
	for _, owner := range adj {
		if owner != nil {
			count[m[owner]]++
			nonEmptyCells++
		}
	}

	// If there are any adjacent non-empty cells, the current cell has a
	// nonEmptyWeight chance of becoming non-empty (divided among the adjacent players
	// in proportion to their representation).
	// TODO give more weight towards remaining the same
	nonEmptyShareSize := 0.0
	if nonEmptyCells > 0.0 {
		nonEmptyShareSize = float64(nonEmptyWeight / nonEmptyCells)
		// If fully surrounded, can't become unowned
		if nonEmptyCells >= 8 {
			nonEmptyShareSize = float64(1.0 / nonEmptyCells)
		}
	}
	l := 0.0
	r := 0.0
	// TODO there's probably a smarter O(1) way to do this
	for i := 0; i < len(g.players); i++ {
		r = l + count[i]*nonEmptyShareSize
		if x >= l && x < r {
			return g.players[i]
		}
		l = r
	}
	return nil // unowned
}

// draw renders all the map entities to the screen via termbox.
func (g *game) draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	// Draw blank map.
	for x := 0; x < g.mapW; x++ {
		for y := 0; y < g.mapH; y++ {
			termbox.SetCell(x, y, ' ', termbox.ColorWhite, termbox.ColorBlack)
		}
	}

	// Draw blobs into back buffer
	for x := range g.next {
		for y, pl := range g.next[x] {
			if pl != nil {
				termbox.SetCell(x, y, pl.symbol, termbox.ColorWhite, pl.color)
			}
		}
	}

	// Draw player into back buffer
	pl := g.players[0] // TODO
	termbox.SetCell(pl.x, pl.y, '@', pl.color, termbox.ColorBlack)

	termbox.Flush()

}

func (g *game) handleEvent(e termbox.Event) {
	p := g.players[0] // TODO eventually we'll have multiple players, and only one will be the local controller
	gameLog.Debug("handling key event", "player", p.name)
	switch e.Key {
	case termbox.KeyArrowRight:
		gameLog.Debug("right key")
		p.updatePos(p.x+1, p.y)
	case termbox.KeyArrowLeft:
		p.updatePos(p.x-1, p.y)
	case termbox.KeyArrowUp:
		p.updatePos(p.x, p.y-1)
	case termbox.KeyArrowDown:
		p.updatePos(p.x, p.y+1)
	default:
		// Undefined key command: noop
	}
}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	rand.Seed(time.Now().UTC().UnixNano())

	// Initialize entities
	g := newGame()

	// Start the input handler thread
	g.input.start()
	defer g.input.stop()

	g.draw()

mainloop:
	for {
		time.Sleep(25 * time.Millisecond) // hacky 40 fps

		// Handle key inputs
		select {
		case e := <-g.input.eventQ:
			if e.Key == g.input.endKey {
				break mainloop
			} else if e.Type == termbox.EventKey {
				gameLog.Debug("received input event")
				g.handleEvent(e)
			}
		default:
			// No input => noop
		}

		g.update()
		g.draw()

		// swap the maps and clear next to prep for next update
		tmp := g.curr
		g.curr = g.next
		g.next = tmp
		g.next.clear()
	}

	gameLog.Info("shutting down")
}
