package main

import (
	"testing"

	"github.com/nsf/termbox-go"
	"github.com/stretchr/testify/assert"
)

func newTestGame(mapW, mapH int) *game {
	g := &game{
		mapW: mapW,
		mapH: mapH,
		curr: newBlobMap(mapW, mapH),
		next: newBlobMap(mapW, mapH),
	}
	g.addPlayer("testplayer0", '0', termbox.ColorGreen, 0, 0)
	return g
}

func TestMax(t *testing.T) {
	assert := assert.New(t)
	expected := 5
	in := []int{1, 2, 3, 4, 5}
	for _, n := range in {
		assert.Equal(expected, max(n, 5))
	}
}

func TestAdjacent(t *testing.T) {
	assert := assert.New(t)
	g := newTestGame(3, 3)

	// empty map
	adj := adjacent(g, 1, 1)
	for i := 0; i < 9; i++ {
		assert.Nil(adj[i])
	}

	// add some blob cells
	p := g.players[0]
	g.curr[0][0] = p // UPLEFT of (1,1)
	g.curr[0][2] = p // UPRIGHT of (1,1)
	g.curr[1][1] = p // CENTER of (1,1)
	adj = adjacent(g, 1, 1)
	for i := 0; i < 9; i++ {
		if i == UPLEFT || i == UPRIGHT || i == CENTER {
			assert.Equal(p, adj[i])
		} else {
			assert.Nil(adj[i])
		}
	}
}

func TestNext(t *testing.T) {
	assert := assert.New(t)
	g := newTestGame(3, 3)

	// empty adjacents means next will be empty
	adj := adjacent(g, 1, 1)
	for i := 0.0; i < 1.0; i += 0.1 {
		assert.Nil(next(g, adj, i))
	}

	// Even one ajacent player means a nonEmptyWeight probability
	// of next being nonEmpty
	p0 := g.players[0]
	g.curr[0][0] = p0 // UPLEFT of (1,1)
	adj = adjacent(g, 1, 1)
	for i := 0.0; i < nonEmptyWeight; i += 0.1 {
		assert.Equal(p0, next(g, adj, i))
	}
	// 1.0 - nonEmptyWeight probability of becoming unowned
	for i := 0.7; i < 1.0; i += 0.1 {
		assert.Nil(next(g, adj, i))
	}

	// Multiple players divy up the nonEmptyWeight probability
	g.addPlayer("testplayer1", '1', termbox.ColorGreen, 0, 0)
	p1 := g.players[1]
	g.curr[0][2] = p1 // UPRIGHT of (1,1)
	adj = adjacent(g, 1, 1)

	// [0.0,0.35) p0, [0.35,0.7) p1, [0.7,1.0) empty
	assert.Equal(p0, next(g, adj, 0.22))
	assert.Equal(p1, next(g, adj, 0.44))
	assert.Nil(next(g, adj, 0.88))
}
