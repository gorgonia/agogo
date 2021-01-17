package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorgonia/agogo/game"
	"github.com/gorilla/websocket"
)

// Encoder is a structure that encodes a game state according to the agogo.OutputEncoder interface
type Encoder struct {
	move        chan game.PlayerMove
	info        chan info
	currentGame int32
}

var upgrader = websocket.Upgrader{} // use default options

func (enc *Encoder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	var b []byte
	for {
		select {
		case move := <-enc.move:
			b, _ = json.Marshal(move)
		case info := <-enc.info:
			b, _ = json.Marshal(info)
		case <-r.Context().Done():
			break
		}
		err = c.WriteMessage(websocket.TextMessage, b)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

// NewEncoder with height and width
func NewEncoder() *Encoder {
	return &Encoder{
		info: make(chan info),
		move: make(chan game.PlayerMove),
	}
}

/*
	repr := `⎢ X · · ⎥
⎢ · · · ⎥
⎢ · · · ⎥`
*/

// Encode a game
func (enc *Encoder) Encode(ms game.MetaState) error {
	g := ms.State()
	log.Println(g.LastMove())
	log.Println(g.Ended())
	enc.move <- g.LastMove()
	if ended, winner := g.Ended(); ended {
		enc.info <- info{
			Epoch:  ms.Epoch(),
			Game:   ms.GameNumber(),
			Winner: winner,
		}
	}
	return nil
}

// Flush ...
func (enc *Encoder) Flush() error { return nil }
