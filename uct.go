// Copyright 2021 go-mcts. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mcts

import (
	"math/rand"
	"time"
)

func computeTree(rootState State, rd *rand.Rand, opts ...Option) *node {
	options := newOptions(opts...)

	if options.MaxIterations < 0 && options.MaxTime < 0 {
		panic("illegal options")
	}

	if rootState.PlayerToMove() != 1 && rootState.PlayerToMove() != 2 {
		panic("only support player1 and player2")
	}

	startTime := time.Now()
	printTime := startTime

	root := newNode(rootState, nil, nil)
	for i := 1; i <= options.MaxIterations || options.MaxIterations < 0; i++ {
		node := root
		state := rootState.Clone()

		for !node.hasUntriedMoves() && node.hasChildren() {
			node = node.selectChildUCT()
			state.DoMove(node.move)
		}

		if node.hasUntriedMoves() {
			move := node.getUntriedMove(rd)
			state.DoMove(move)
			node = node.addChild(move, state)
		}

		for state.HasMoves() {
			state.DoRandomMove(rd)
		}

		for node != nil {
			node.update(state.GetResult(node.playerToMove))
			node = node.parent
		}

		if options.Verbose || options.MaxTime >= 0 {
			now := time.Now()
			if options.Verbose && (now.Sub(printTime) >= time.Second || i == options.MaxIterations) {
				Debugf("%d games played (%.2f / second).", i, float64(i)/now.Sub(startTime).Seconds())
				printTime = now
			}

			if options.MaxTime >= 0 && now.Sub(startTime) >= options.MaxTime {
				break
			}
		}
	}

	return root
}

func ComputeMove(rootState State, opts ...Option) Move {
	options := newOptions(opts...)

	if rootState.PlayerToMove() != 1 && rootState.PlayerToMove() != 2 {
		panic("only support player1 and player2")
	}

	moves := rootState.GetMoves()
	if len(moves) == 0 {
		panic("root moves is empty")
	}

	if len(moves) == 1 {
		return moves[0]
	}

	startTime := time.Now()

	rootFutures := make(chan *node, options.Groutines)
	for i := 0; i < options.Groutines; i++ {
		go func() {
			rd := rand.New(rand.NewSource(time.Now().UnixNano()))
			rootFutures <- computeTree(rootState, rd, opts...)
		}()
	}

	visits := make(map[Move]int)
	wins := make(map[Move]float64)
	gamePlayed := 0
	for i := 0; i < options.Groutines; i++ {
		root := <-rootFutures
		gamePlayed += root.visits
		for _, c := range root.children {
			visits[c.move] += c.visits
			wins[c.move] += c.wins
		}
	}

	bestScore := float64(-1)
	var bestMove Move
	for move, v := range visits {
		w := wins[move]
		expectedSuccessRate := (w + 1) / (float64(v) + 2)
		if expectedSuccessRate > bestScore {
			bestMove = move
			bestScore = expectedSuccessRate
		}

		if options.Verbose {
			Debugf("Move: %v (%2d%% visits) (%2d%% wins)",
				move, int(100.0*float64(v)/float64(gamePlayed)+0.5), int(100.0*w/float64(v)+0.5))
		}
	}

	if options.Verbose {
		bestWins := wins[bestMove]
		bestVisits := visits[bestMove]
		Debugf("Best: %v (%2d%% visits) (%2d%% wins)",
			bestMove,
			int(100.0*float64(bestVisits)/float64(gamePlayed)+0.5),
			int(100.0*bestWins/float64(bestVisits)+0.5),
		)

		now := time.Now()
		Debugf(
			"%d games played in %.2f s. (%.2f / second, %d parallel jobs).",
			gamePlayed,
			now.Sub(startTime).Seconds(),
			float64(gamePlayed)/now.Sub(startTime).Seconds(),
			options.Groutines,
		)
	}
	return bestMove
}
