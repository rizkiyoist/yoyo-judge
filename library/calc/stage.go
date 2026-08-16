/*
 * Created on Thu Jun 19 2025
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package calc

// ScoringStage is which IYYF scoring workbook variant a contest follows:
// FINAL (ver. 2.3, 2017-8-3 Hironori Mii) uses 8 evaluation categories each
// halved before summing; PRELIM/Semi-Final uses 4 categories summed directly.
type ScoringStage string

const (
	StageFinal  ScoringStage = "final"
	StagePrelim ScoringStage = "prelim"
)

// DefaultClickerValue is the SET-UP "Clicker Value" (T.Ex max), unchanged
// between the FINAL and PRELIM workbooks.
const DefaultClickerValue = 60

// FinalCategories mirrors the FINAL workbook's SET-UP "Given Value 1-8"
// rows: EXE/CTL/TDV/SEM roll up into T.Ev, MU1/MU2/BDY/SHW into P.Ev, and
// FINAL-SCORE halves each averaged raw score before summing.
func FinalCategories() []EvalCategory {
	return []EvalCategory{
		{Name: "EXE", Group: GroupTechnicalEval, MaxValue: 5, Halve: true},
		{Name: "CTL", Group: GroupTechnicalEval, MaxValue: 5, Halve: true},
		{Name: "TDV", Group: GroupTechnicalEval, MaxValue: 5, Halve: true},
		{Name: "SEM", Group: GroupTechnicalEval, MaxValue: 5, Halve: true},
		{Name: "MU1", Group: GroupPerformanceEval, MaxValue: 5, Halve: true},
		{Name: "MU2", Group: GroupPerformanceEval, MaxValue: 5, Halve: true},
		{Name: "BDY", Group: GroupPerformanceEval, MaxValue: 5, Halve: true},
		{Name: "SHW", Group: GroupPerformanceEval, MaxValue: 5, Halve: true},
	}
}

// FinalDeductions mirrors the FINAL workbook's SET-UP "Deduction 1-3" rows.
func FinalDeductions() []Deduction {
	return []Deduction{
		{Name: "Stop", Value: 1},
		{Name: "Discard", Value: 3},
		{Name: "Cut", Value: 5},
	}
}

// PrelimCategories mirrors the PRELIM/Semi-Final workbook's SET-UP "Given
// Value 1-4" rows: EXE/CTL roll up into T.Ev, MU1/BDY into P.Ev, and
// FINAL-SCORE sums the averaged raw scores directly (no halving).
func PrelimCategories() []EvalCategory {
	return []EvalCategory{
		{Name: "EXE", Group: GroupTechnicalEval, MaxValue: 10, Halve: false},
		{Name: "CTL", Group: GroupTechnicalEval, MaxValue: 10, Halve: false},
		{Name: "MU1", Group: GroupPerformanceEval, MaxValue: 10, Halve: false},
		{Name: "BDY", Group: GroupPerformanceEval, MaxValue: 10, Halve: false},
	}
}

// PrelimDeductions mirrors the PRELIM/Semi-Final workbook's SET-UP
// "Deduction 1-3" rows (the third deduction is named "Detach" instead of
// "Cut", but behaves identically).
func PrelimDeductions() []Deduction {
	return []Deduction{
		{Name: "Stop", Value: 1},
		{Name: "Discard", Value: 3},
		{Name: "Detach", Value: 5},
	}
}

// NewContest builds a Contest pre-populated with the given stage's default
// categories, deductions and clicker value.
func NewContest(stage ScoringStage, players []PlayerInput) *Contest {
	c := &Contest{
		Stage:        stage,
		ClickerValue: DefaultClickerValue,
		Players:      players,
	}
	switch stage {
	case StagePrelim:
		c.Categories = PrelimCategories()
		c.Deductions = PrelimDeductions()
	default:
		c.Categories = FinalCategories()
		c.Deductions = FinalDeductions()
	}
	return c
}
