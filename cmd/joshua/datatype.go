package main

import "github.com/gorgonia/agogo/game"

const (
	kindPlay = 0
	kindInfo = 1
)

type info struct {
	Epoch  int
	Game   int
	Winner game.Player
}
