/*
 * Created on Sun Aug 16 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

// Package server is the HTTP API for the yoyo-judge frontend's ScoringApi
// contract (frontend/src/api/client.ts). Backed by SQLite via GORM (store.go);
// Google OAuth is handled in oauth.go.
package server

import "yoyo-judge/library/calc"

// User mirrors frontend/src/types.ts's User.
type User struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

// Division mirrors frontend/src/types.ts's Division.
type Division struct {
	ID        string              `json:"id"`
	ContestID string              `json:"contestId"`
	Name      string              `json:"name"`
	Stages    []calc.ScoringStage `json:"stages"`
}

// Contest mirrors frontend/src/types.ts's Contest.
type Contest struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Year            int        `json:"year"`
	OwnerUserID     string     `json:"ownerUserId"`
	HeadJudgeUserID string     `json:"headJudgeUserId"`
	Locked          bool       `json:"locked"`
	Divisions       []Division `json:"divisions"`
}

// JudgeRole mirrors frontend/src/types.ts's JudgeRole.
type JudgeRole string

const (
	RoleClicker   JudgeRole = "clicker"
	RoleEvaluator JudgeRole = "evaluator"
)

// JudgeAssignment mirrors frontend/src/types.ts's JudgeAssignment.
type JudgeAssignment struct {
	ID         string            `json:"id"`
	ContestID  string            `json:"contestId"`
	DivisionID string            `json:"divisionId"`
	Stage      calc.ScoringStage `json:"stage"`
	UserID     string            `json:"userId"`
	Role       JudgeRole         `json:"role"`
	Slot       int               `json:"slot"`
}

// Player mirrors frontend/src/types.ts's Player.
type Player struct {
	ID         string `json:"id"`
	DivisionID string `json:"divisionId"`
	Number     int    `json:"number"`
	Name       string `json:"name"`
}

// ClickerInput mirrors frontend/src/types.ts's ClickerInput.
type ClickerInput struct {
	Plus  int `json:"plus"`
	Minus int `json:"minus"`
}

// MajorDeductions mirrors frontend/src/types.ts's MajorDeductions.
type MajorDeductions struct {
	Stop    int `json:"stop"`
	Discard int `json:"discard"`
	Cut     int `json:"cut"`
}

// PlayerRawScores mirrors frontend/src/types.ts's PlayerRawScores: one
// player's raw judge inputs for one division+stage, keyed by judge slot
// (1-6).
type PlayerRawScores struct {
	PlayerID   string                     `json:"playerId"`
	Clickers   map[int]ClickerInput       `json:"clickers"`
	Deductions MajorDeductions            `json:"deductions"`
	Evals      map[int]map[string]float64 `json:"evals"`
}

// PlayerResultResponse adds the playerId the frontend expects onto
// library/calc's PlayerResult, without modifying that package.
type PlayerResultResponse struct {
	PlayerID           string             `json:"playerId"`
	Number             int                `json:"number"`
	Name               string             `json:"name"`
	TechnicalExecution float64            `json:"technicalExecution"`
	CategoryScores     map[string]float64 `json:"categoryScores"`
	GroupTotals        map[string]float64 `json:"groupTotals"`
	EvaluationTotal    float64            `json:"evaluationTotal"`
	DeductionTotals    map[string]float64 `json:"deductionTotals"`
	DeductionTotal     float64            `json:"deductionTotal"`
	FinalScore         float64            `json:"finalScore"`
	Place              int                `json:"place"`
}

func toPlayerResultResponse(playerID string, r calc.PlayerResult) PlayerResultResponse {
	groupTotals := make(map[string]float64, len(r.GroupTotals))
	for group, total := range r.GroupTotals {
		groupTotals[string(group)] = total
	}
	return PlayerResultResponse{
		PlayerID:           playerID,
		Number:             r.Number,
		Name:               r.Name,
		TechnicalExecution: r.TechnicalExecution,
		CategoryScores:     r.CategoryScores,
		GroupTotals:        groupTotals,
		EvaluationTotal:    r.EvaluationTotal,
		DeductionTotals:    r.DeductionTotals,
		DeductionTotal:     r.DeductionTotal,
		FinalScore:         r.FinalScore,
		Place:              r.Place,
	}
}
