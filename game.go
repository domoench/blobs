package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mgutz/logxi/v1"
	"github.com/nsf/termbox-go"
)

// TODO
//  - move things into their own files

var gameLog log.Logger

const (
	mapWidth  = 80
	mapHeight = 50
)

type player struct {
	name   string
	symbol rune // Player's blob symbol
	blobs  []*blob
	g      *game
	point
}

func (p *player) updatePos(x, y int) {
	if x >= 0 && x < mapWidth {
		p.x = x
	}
	if y >= 0 && y < mapHeight {
		p.y = y
	}
}

type point struct {
	x, y int
}

func (p point) String() string {
	return fmt.Sprintf("(%d,%d)", p.x, p.y)
}

type pointset map[point]bool

func newPointSet() pointset {
	return make(map[point]bool)
}

func (ps pointset) contains(p point) bool {
	_, found := ps[p]
	return found
}

// add adds the point to the pointset
func (ps pointset) add(p point) {
	ps[p] = true
}

func (ps pointset) remove(p point) {
	delete(ps, p)
}

// Directions: indexes into 9-element adjacent point slices
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

// adjPoints lists the 9 points adjacent to and including
// the cell at point p
func adjPoints(p point) []point {
	adj := make([]point, 9)
	i := 0
	for y := p.y - 1; y <= p.y+1; y++ {
		for x := p.x - 1; x <= p.x+1; x++ {
			adj[i] = point{x, y}
			i++
		}
	}
	return adj
}

type blob struct {
	// TODO add unique ID (for recognizing blob merges)
	points   pointset
	boundary pointset // A set of cells that form the boundary to the set of cells exterior to this blob
	overlord *player
}

// add adds the point to this blobs points, and updates the boundary accordingly
func (b *blob) add(p point) {
	b.points.add(p)

	// TODO this is a roundabout way to access the game struct. Is there a cleaner way that
	// isn't just a globally accessable game reference?
	g := b.overlord.g

	// recalculate boundary set
	// TODO Handle blob merges (same owner, diff blob)
	adj := adjPoints(p)
	for _, adjPoint := range adj {
		pointOwner := g.memberOf(adjPoint)
		gameLog.Debug("adj",
			"point", p,
			"adjPoint", adjPoint,
			"blob", b,
			"adjBlob", pointOwner,
		)
		if g.memberOf(adjPoint) != b {
			b.boundary.add(adjPoint)
		}
	}
}

// Game maintains game state
type game struct {
	players []*player
	input   *input
}

// addBlob starts a blob at the given point
func (pl *player) addBlob(p point) {
	b := &blob{
		points:   newPointSet(),
		boundary: newPointSet(),
		overlord: pl,
	}
	pl.blobs = append(pl.blobs, b)
	b.add(p)
}

// addPlayer adds a new player (with no blobs) to the game
func (g *game) addPlayer(name string, symbol rune, start point) {
	p := &player{name: name, symbol: symbol, g: g, point: start}
	g.players = append(g.players, p)
}

// newGame initializes the game's state
func newGame() *game {
	gameLog.Info("initializing new game...")

	g := game{input: newInput()}
	g.addPlayer("david", 'B', point{0, 0})

	// Start players with random blob seeds
	blobsPerPlayer := 1
	for _, p := range g.players {
		// TODO: Random walk generate the blobs
		for i := 0; i < blobsPerPlayer; i++ {
			p.addBlob(point{rand.Intn(mapWidth), rand.Intn(mapHeight)})
		}
	}

	return &g
}

// memberOf returns a reference to the blob this point belongs to, nil if unowned
func (g *game) memberOf(p point) *blob {
	gameLog.Debug("memberOf()")
	for _, pl := range g.players {
		for _, b := range pl.blobs {
			if b.points.contains(p) {
				return b
			}
		}
	}

	// otherwise this blob is unowned
	return nil
}

// update calculates all the entity interactions since the last loop
// iteration and updates their states accordingly.
func (g *game) update() {
	// Calculate blob growth
	for _, pl := range g.players {
		for _, b := range pl.blobs {
			boundaryPoints := []point{}
			for p := range b.boundary {
				boundaryPoints = append(boundaryPoints, p)
			}

			// expand the blob into one of its boundary cells
			p := boundaryPoints[rand.Intn(len(boundaryPoints))]
			b.boundary.remove(p)
			b.add(p)
		}
	}
}

// draw renders all the map entities to the screen via termbox.
func (g *game) draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	// Draw blank map
	for x := 0; x < mapWidth; x++ {
		for y := 0; y < mapHeight; y++ {
			termbox.SetCell(x, y, ' ', termbox.ColorWhite, termbox.ColorBlack)
		}
	}

	// Draw blobs into back buffer
	for _, p := range g.players {
		for i, b := range p.blobs {
			// TODO b.draw()?
			gameLog.Debug(fmt.Sprintf("drawing blob %d", i), "mass", len(b.points))
			for c, _ := range b.points {
				termbox.SetCell(c.x, c.y, b.overlord.symbol, termbox.ColorWhite, termbox.ColorGreen)
			}

			// Draw boundaries
			for c, _ := range b.boundary {
				termbox.SetCell(c.x, c.y, '*', termbox.ColorWhite, termbox.ColorRed)
			}
		}
	}

	// Draw player into back buffer
	pl := g.players[0] // TODO
	gameLog.Debug("drawing player", "pos", pl.point)
	termbox.SetCell(pl.x, pl.y, '@', termbox.ColorWhite, termbox.ColorBlack)

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

	gameLog = log.NewLogger(os.Stderr, "game")

	rand.Seed(time.Now().UTC().UnixNano())

	// Initialize entities
	g := newGame()

	// Start the input handler thread
	g.input.start()
	defer g.input.stop()

	g.draw()

mainloop:
	for {
		//time.Sleep(50 * time.Millisecond) // hacky 20 fps
		time.Sleep(1 * time.Second)

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
	}

	gameLog.Info("shutting down")
}
