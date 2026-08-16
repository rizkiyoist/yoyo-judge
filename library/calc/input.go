/*
 * Created on Thu Jun 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package calc

// EvalGroup is one of the two evaluation buckets a scored category rolls up
// into: Technical Evaluation (cleanliness/control/diversity/execution-style
// criteria) or Performance Evaluation (music/showmanship/body-control style
// criteria).
type EvalGroup string

const (
	GroupTechnicalEval   EvalGroup = "TEv"
	GroupPerformanceEval EvalGroup = "PEv"
)

// EvalCategory describes one judged category (e.g. "EXE", "CTL"), mirroring
// a "Given Value" row on the SET-UP sheet: its max raw value, which group it
// rolls up into, and whether the stage halves the averaged raw score before
// summing it into the total (the FINAL stage does, PRELIM does not).
type EvalCategory struct {
	Name     string
	Group    EvalGroup
	MaxValue float64
	Halve    bool
}

// Deduction is one major-deduction category (Stop/Discard/Cut or Detach),
// worth Value points per occurrence.
type Deduction struct {
	Name  string
	Value float64
}

// ClickerScore is one technical (clicker) judge's raw plus/minus tally for a
// single player.
type ClickerScore struct {
	Plus  int
	Minus int
}

func (c ClickerScore) Net() int {
	return c.Plus - c.Minus
}

// PlayerInput is one player's raw judge input: the 6 clicker judges'
// plus/minus tallies, the 6 evaluation judges' raw scores per category, and
// the counts of each major deduction incurred.
type PlayerInput struct {
	Number     int
	Name       string
	Clickers   [6]ClickerScore
	EvalScores [6]map[string]float64
	Deductions map[string]int
}

// Contest is a full scoring session: the stage's configuration (clicker
// value, judged categories, deductions) plus every player's raw input.
type Contest struct {
	Stage        ScoringStage
	ClickerValue float64
	Categories   []EvalCategory
	Deductions   []Deduction
	Players      []PlayerInput
}
