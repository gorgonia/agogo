package komi

import (
	"fmt"

	"github.com/gorgonia/agogo/game"
)

type moveError game.PlayerMove

func (err moveError) Error() string {
	return fmt.Sprintf("Unable to make %v", game.PlayerMove(err))
}
