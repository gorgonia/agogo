package gtp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_General(t *testing.T) {
	assert := assert.New(t)
	e := New(nil, "xx", "1", nil)
	var x string

	ch, ret := e.Start()
	ch <- "version"
	x = <-ret
	assert.Equal("= 1\n\n", x)

	ch <- "known_command hello"
	x = <-ret
	assert.Equal("= false\n\n", x)

	ch <- "known_command name"
	x = <-ret
	assert.Equal("= true\n\n", x)

	ch <- "completelyUnheardOfCommand xxx"
	x = <-ret
	assert.Equal("? Unknown command \"completelyunheardofcommand\"\n\n", x)

}
