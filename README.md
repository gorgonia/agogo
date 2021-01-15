# agogo

A reimplementation of AlphaGo in Go (specifically AlphaZero)

## About

The algorithm is composed of:

- a Monte-Carlo Tree Search (MCTS) implemented in the [`mcts`](https://pkg.go.dev/github.com/gorgonia/agogo/mcts) package;
- a Dual Neural Network (DNN) implemented in the [`dualnet`](https://pkg.go.dev/github.com/gorgonia/agogo/dualnet) package.

The algorithm is wrapped into a top-level structure ([`AZ`](https://pkg.go.dev/github.com/gorgonia/agogo#AZ) for AlphaZero). The algorithm applies to any game able to fulfill a specified contract.

The contract specifies the description of a game state.

In this package, the contract is a Go interface declared in the `game` package: [`State`](https://pkg.go.dev/github.com/gorgonia/agogo/game#State).

### Description of some concepts/ubiquitous language

- In the `agogo` package, each player of the game is an [`Agent`](https://pkg.go.dev/github.com/gorgonia/agogo#Agent), and in a `game`, two `Agents` are playing in an [`Arena`](https://pkg.go.dev/github.com/gorgonia/agogo@v0.1.0#Arena)

- The `game` package is loosely coupled with the AlphaZero algorithm and describes a game's behavior (and not what a game is). The behavior is expressed as a set of functions to operate on a [`State`](https://pkg.go.dev/github.com/gorgonia/agogo/game#State) of the game. A State is an interface that represents the current game state *as well* as the allowed interactions. The interaction is made by an object [`Player`](https://pkg.go.dev/github.com/gorgonia/agogo/game#Player) who is operating a [`PlayerMove`](https://pkg.go.dev/github.com/gorgonia/agogo/game#PlayerMove). The implementer's responsibility is to code the game's rules by creating an object that fulfills the State contract and implements the allowed moves.

### Training process

### Applying the Algo on a game

This package is designed to be extensible. Therefore you can train AlphaZero on any board game respecting the contract of the `game` package.
Then, the model can be saved and used as a player.

The steps to train the algorithm are:

- Creating a structure that is fulfilling the [`State`](https://pkg.go.dev/github.com/gorgonia/agogo/game#State) interface (aka a _game_).
- Creating a _configuration_ for your AZ internal MCTS and NN.
- Creating an `AZ` structure based on the _game_ and  the _configuration_
- Executing the learning process (by calling the [`Learn`](https://pkg.go.dev/github.com/gorgonia/agogo#AZ.Learn) method)
- Saving the trained model (by calling the [`Save`](https://pkg.go.dev/github.com/gorgonia/agogo#AZ.Save) method)

The steps to play against the algorithm are:

- Creating an `AZ` object
- Loading the trained model (by calling the [`Read`](https://pkg.go.dev/github.com/gorgonia/agogo#AZ.Read) method))
- TODO: how to create a human player in a game

## Examples

Four board games are implemented so far. Each of them is defined as a subpackage of `game`:

- [`mnk`](https://pkg.go.dev/github.com/gorgonia/agogo/game/mnk) for [m,n,k](https://en.wikipedia.org/wiki/M,n,k-game) game.
- [`wq`](https://pkg.go.dev/github.com/gorgonia/agogo/game/mnk) is the game of [Go](https://en.wikipedia.org/wiki/Go_(game)) (围碁)
- `c4`
- `komi`

### tic-tac-toe

Tic-tac-toe is a m,n,k game where m=n=k=3.

Here is a sample code that trains AlphaGo to play the game. The result is saved in a file `example.model`

```go
// encodeBoard is a GameEncoder (https://pkg.go.dev/github.com/gorgonia/agogo#GameEncoder) for the tic-tac-toe
func encodeBoard(a game.State) []float32 {
	board := agogo.EncodeTwoPlayerBoard(a.Board(), nil)
	for i := range board {
		if board[i] == 0 {
			board[i] = 0.001
		}
	}
	playerLayer := make([]float32, len(a.Board()))
	next := a.ToMove()
	if next == game.Player(game.Black) {
		for i := range playerLayer {
			playerLayer[i] = 1
		}
	} else if next == game.Player(game.White) {
		// vecf32.Scale(board, -1)
		for i := range playerLayer {
			playerLayer[i] = -1
		}
	}
	retVal := append(board, playerLayer...)
	return retVal
}

func main() {
    // Create the configuration of the neural network
	conf := agogo.Config{
		Name:            "Tic Tac Toe",
		NNConf:          dual.DefaultConf(3, 3, 10),
		MCTSConf:        mcts.DefaultConfig(3),
		UpdateThreshold: 0.52,
	}
	conf.NNConf.BatchSize = 100
	conf.NNConf.Features = 2 // write a better encoding of the board, and increase features (and that allows you to increase K as well)
	conf.NNConf.K = 3
	conf.NNConf.SharedLayers = 3
	conf.MCTSConf = mcts.Config{
		PUCT:           1.0,
		M:              3,
		N:              3,
		Timeout:        100 * time.Millisecond,
		PassPreference: mcts.DontPreferPass,
		Budget:         1000,
		DumbPass:       true,
		RandomCount:    0,
	}

	conf.Encoder = encodeBoard

    // Create a new game
    g := mnk.TicTacToe()
    // Create the AlphaZero structure 
    a := agogo.New(g, conf)
    // Launch the learning process
    a.Learn(5, 30, 200, 30) // 5 epochs, 50 episode, 100 NN iters, 100 games.
    // Save the model
	a.Save("example.model")
}
```
