/*
 * Created on Thu Jun 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package calc

import (
	"math"
	"testing"
)

func evalScores(exe, ctl, tdv, sem, mu1, mu2, bdy, shw float64) [6]map[string]float64 {
	scores := map[string]float64{
		"EXE": exe, "CTL": ctl, "TDV": tdv, "SEM": sem,
		"MU1": mu1, "MU2": mu2, "BDY": bdy, "SHW": shw,
	}
	var out [6]map[string]float64
	for i := range out {
		out[i] = scores
	}
	return out
}

func clickers(nets ...int) [6]ClickerScore {
	var out [6]ClickerScore
	for i, n := range nets {
		out[i] = ClickerScore{Plus: n, Minus: 0}
	}
	return out
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestCalculateFinalStage(t *testing.T) {
	players := []PlayerInput{
		{
			Number:     1,
			Name:       "A",
			Clickers:   clickers(10, 12, 11, 9, 10, 11),
			EvalScores: evalScores(4, 4, 4, 4, 4, 4, 4, 4),
			Deductions: map[string]int{"Stop": 1},
		},
		{
			Number:     2,
			Name:       "B",
			Clickers:   clickers(12, 14, 13, 10, 12, 13),
			EvalScores: evalScores(4.5, 4.5, 4.5, 4.5, 4.5, 4.5, 4.5, 4.5),
			Deductions: map[string]int{},
		},
	}

	contest := NewContest(StageFinal, players)
	results := contest.Calculate()

	a, b := results[0], results[1]

	if !almostEqual(a.TechnicalExecution, 51.16117216117215) {
		t.Errorf("A TechnicalExecution = %v, want ~51.1612", a.TechnicalExecution)
	}
	if !almostEqual(b.TechnicalExecution, 60.0) {
		t.Errorf("B TechnicalExecution = %v, want 60", b.TechnicalExecution)
	}

	if !almostEqual(a.GroupTotals[GroupTechnicalEval], 8.0) {
		t.Errorf("A T.Ev total = %v, want 8", a.GroupTotals[GroupTechnicalEval])
	}
	if !almostEqual(a.GroupTotals[GroupPerformanceEval], 8.0) {
		t.Errorf("A P.Ev total = %v, want 8", a.GroupTotals[GroupPerformanceEval])
	}
	if !almostEqual(b.GroupTotals[GroupTechnicalEval], 9.0) {
		t.Errorf("B T.Ev total = %v, want 9", b.GroupTotals[GroupTechnicalEval])
	}

	if !almostEqual(a.EvaluationTotal, 67.16117216117215) {
		t.Errorf("A EvaluationTotal = %v, want ~67.1612", a.EvaluationTotal)
	}
	if !almostEqual(b.EvaluationTotal, 78.0) {
		t.Errorf("B EvaluationTotal = %v, want 78", b.EvaluationTotal)
	}

	if !almostEqual(a.DeductionTotal, 1.0) {
		t.Errorf("A DeductionTotal = %v, want 1", a.DeductionTotal)
	}
	if !almostEqual(a.FinalScore, 66.16117216117215) {
		t.Errorf("A FinalScore = %v, want ~66.1612", a.FinalScore)
	}
	if !almostEqual(b.FinalScore, 78.0) {
		t.Errorf("B FinalScore = %v, want 78", b.FinalScore)
	}

	if a.Place != 2 {
		t.Errorf("A Place = %d, want 2", a.Place)
	}
	if b.Place != 1 {
		t.Errorf("B Place = %d, want 1", b.Place)
	}
}

func TestCalculateTiedPlacesSkipRank(t *testing.T) {
	players := []PlayerInput{
		{Number: 1, Clickers: clickers(10, 10, 10, 10, 10, 10), EvalScores: evalScores(4, 4, 4, 4, 4, 4, 4, 4)},
		{Number: 2, Clickers: clickers(10, 10, 10, 10, 10, 10), EvalScores: evalScores(4, 4, 4, 4, 4, 4, 4, 4)},
		{Number: 3, Clickers: clickers(5, 5, 5, 5, 5, 5), EvalScores: evalScores(2, 2, 2, 2, 2, 2, 2, 2)},
	}
	results := NewContest(StageFinal, players).Calculate()

	if results[0].Place != 1 || results[1].Place != 1 {
		t.Fatalf("expected players 1 and 2 tied at place 1, got %d and %d", results[0].Place, results[1].Place)
	}
	if results[2].Place != 3 {
		t.Fatalf("expected player 3 at place 3 (rank skips over the tie), got %d", results[2].Place)
	}
}

func TestCalculatePrelimStageNoHalving(t *testing.T) {
	players := []PlayerInput{
		{
			Number:     1,
			Clickers:   clickers(10, 10, 10, 10, 10, 10),
			EvalScores: evalScores(8, 8, 0, 0, 8, 0, 8, 0),
		},
	}
	results := NewContest(StagePrelim, players).Calculate()
	r := results[0]

	if !almostEqual(r.CategoryScores["EXE"], 8.0) {
		t.Errorf("EXE score = %v, want 8 (prelim does not halve)", r.CategoryScores["EXE"])
	}
	if !almostEqual(r.GroupTotals[GroupTechnicalEval], 16.0) {
		t.Errorf("T.Ev total = %v, want 16", r.GroupTotals[GroupTechnicalEval])
	}
	if !almostEqual(r.GroupTotals[GroupPerformanceEval], 16.0) {
		t.Errorf("P.Ev total = %v, want 16", r.GroupTotals[GroupPerformanceEval])
	}
}
