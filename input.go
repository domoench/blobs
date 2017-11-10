package main

// Based on https://github.com/JoelOtter/termloop

import (
	"os"

	"github.com/mgutz/logxi/v1"
	"github.com/nsf/termbox-go"
)

var inputLog log.Logger

type input struct {
	endKey termbox.Key
	eventQ chan termbox.Event
	ctrl   chan bool
}

func newInput() *input {
	inputLog = log.NewLogger(os.Stderr, "input")
	i := input{eventQ: make(chan termbox.Event),
		ctrl:   make(chan bool, 2),
		endKey: termbox.KeyCtrlC}
	return &i
}

func (i *input) start() {
	go poll(i)
}

func (i *input) stop() {
	inputLog.Debug("stopping polling")
	i.ctrl <- true
}

func poll(i *input) {
	inputLog.Debug("starting polling")
loop:
	for {
		select {
		case <-i.ctrl:
			break loop
		default:
			i.eventQ <- termbox.PollEvent()
		}
	}
}
