// Copyright 2021 go-mcts. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mcts

import (
	"math/rand"
	"time"

	"github.com/go-mcts/mcts/internal/log"
)

type Score struct {
	Move	Move
	Wins	float64
	Visits	float64
}

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

		now := time.Now()
		if now.Sub(printTime) >= time.Second || i == options.MaxIterations {
			log.Debugf("%d games played (%.2f / second).", i, float64(i)/now.Sub(startTime).Seconds())
			printTime = now
		}

		if options.MaxTime >= 0 && now.Sub(startTime) >= options.MaxTime {
			break
		}
	}

	return root
}

func ComputeMove(rootState State, opts ...Option) []Score {
	options := newOptions(opts...)

	if rootState.PlayerToMove() != 1 && rootState.PlayerToMove() != 2 {
		panic("only support player1 and player2")
	}

	moves := rootState.GetMoves()
	if len(moves) == 0 {
		panic("root moves is empty")
	}

	if len(moves) == 1 {
		return []Score{Score{
			Move: moves[0],
			Wins: 0,
			Visits: 0,
		}}
	}

	rootFutures := make(chan *node, options.Goroutines)
	for i := 0; i < options.Goroutines; i++ {
		go func() {
			rd := rand.New(rand.NewSource(time.Now().UnixNano()))
			rootFutures <- computeTree(rootState, rd, opts...)
		}()
	}

	visits := newCounter()
	wins := newCounter()
	gamePlayed := 0
	for i := 0; i < options.Goroutines; i++ {
		root := <-rootFutures
		gamePlayed += root.visits
		for _, c := range root.children {
			visits.incr(c.move, float64(c.visits))
			wins.incr(c.move, c.wins)
		}
	}

	var scores []Score
	visits.rng(func(key interface{}, v float64) {
		move := key.(Move)
		w := wins.get(move)
		score := Score{
			Move: move,
			Wins: w,
			Visits: v,
		}
		scores = append(scores, score)

	})

	return scores
}

