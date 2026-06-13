package models

import (
	"math/rand"
	"time"
)

// AIDifficulty represents the AI opponent profiles
type AIDifficulty string

const (
	AIDifficultyEasy   AIDifficulty = "easy"   // Rando-tron
	AIDifficultyMedium AIDifficulty = "medium" // Cycle-bot
	AIDifficultyHard   AIDifficulty = "hard"   // Predictor (Markov Chain)
)

// AIEngine handles prediction calculations and pattern tracking
type AIEngine struct {
	Difficulty     AIDifficulty
	History        []Move                // Sequence of moves the player has chosen
	TransitionMap  map[Move]map[Move]int // Transition counts: [PrevMove][NextMove]Count
	lastAIMove     Move
	rng            *rand.Rand
}

// NewAIEngine creates a new initialized AI opponent
func NewAIEngine(difficulty AIDifficulty) *AIEngine {
	return &AIEngine{
		Difficulty:    difficulty,
		History:       []Move{},
		TransitionMap: make(map[Move]map[Move]int),
		lastAIMove:    None,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())), // #nosec G404 - game AI does not need crypto RNG
	}
}

// RecordPlayerMove logs the player's choice to update the Markov transition map
func (ae *AIEngine) RecordPlayerMove(playerMove Move) {
	ae.History = append(ae.History, playerMove)

	// We need at least 2 moves to establish a transition pattern (e.g. Rock -> Paper)
	if len(ae.History) < 2 {
		return
	}

	prevMove := ae.History[len(ae.History)-2]
	if _, exists := ae.TransitionMap[prevMove]; !exists {
		ae.TransitionMap[prevMove] = make(map[Move]int)
	}
	ae.TransitionMap[prevMove][playerMove]++
}

// PredictPlayerNextMove determines the player's most likely next choice using history
func (ae *AIEngine) PredictPlayerNextMove() Move {
	if len(ae.History) == 0 {
		return None
	}

	lastMove := ae.History[len(ae.History)-1]
	transitions, ok := ae.TransitionMap[lastMove]
	if !ok || len(transitions) == 0 {
		return None // No transition data for this state yet
	}

	// Find the move with the highest occurrence count
	var predictedMove Move
	maxCount := -1
	for move, count := range transitions {
		if count > maxCount {
			maxCount = count
			predictedMove = move
		}
	}
	return predictedMove
}

// GetCounterMove returns the winning counter to a given move
func GetCounterMove(move Move) Move {
	switch move {
	case Rock:
		return Paper
	case Paper:
		return Scissors
	case Scissors:
		return Rock
	default:
		return Rock
	}
}

// ChooseMove calculates the AI's move based on the selected difficulty profile
func (ae *AIEngine) ChooseMove() Move {
	moves := []Move{Rock, Paper, Scissors}

	switch ae.Difficulty {
	case AIDifficultyEasy:
		// Rando-tron: Pure random
		return moves[ae.rng.Intn(3)]

	case AIDifficultyMedium:
		// Cycle-bot: Rocks, Papers, Scissors cycle
		switch ae.lastAIMove {
		case Rock:
			ae.lastAIMove = Paper
		case Paper:
			ae.lastAIMove = Scissors
		case Scissors:
			ae.lastAIMove = Rock
		default:
			ae.lastAIMove = Rock
		}
		return ae.lastAIMove

	case AIDifficultyHard:
		// Predictor: Markov Chain
		predicted := ae.PredictPlayerNextMove()
		if predicted == None {
			// Fall back to random if we have no pattern data yet
			return moves[ae.rng.Intn(3)]
		}
		return GetCounterMove(predicted)

	default:
		return moves[ae.rng.Intn(3)]
	}
}
