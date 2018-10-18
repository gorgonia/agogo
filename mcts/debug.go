// +build debug

package mcts

import (
	"bytes"
	"fmt"
)

type logstruct struct {
	msg  string
	args []interface{}
}

type lumberjack struct {
	*bytes.Buffer
	ch chan logstruct
}

func makeLumberJack() lumberjack {
	return lumberjack{
		Buffer: new(bytes.Buffer),
		ch:     make(chan logstruct),
	}
}

func (l *lumberjack) start() {
	for s := range l.ch {
		fmt.Fprintf(l.Buffer, s.msg, s.args...)
		l.WriteByte('\n')
	}
}

func (l *lumberjack) log(msg string, args ...interface{}) {
	l.ch <- logstruct{msg: msg, args: args}
}

func (l *lumberjack) Reset() { l.Buffer.Reset() }

func (l lumberjack) Log() string { return l.String() }
