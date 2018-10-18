// +build !debug

package mcts

type lumberjack struct{}

func makeLumberJack() lumberjack { return lumberjack{} }

func (l lumberjack) start() {}

func (l lumberjack) log(msg string, args ...interface{}) {}

func (l lumberjack) Log() string { return "" }

func (l lumberjack) Reset() {}
