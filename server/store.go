/*
 * Created on Sun Aug 16 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"yoyo-judge/library/calc"
)

var ErrNotFound = errors.New("not found")

func newID(prefix string) string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf))
}

type divisionRecord struct {
	Division
	Players     []Player
	Assignments []JudgeAssignment
	Scores      map[calc.ScoringStage][]PlayerRawScores
}

type contestRecord struct {
	ID          string
	Name        string
	OwnerUserID string
	Divisions   []*divisionRecord
}

func (c *contestRecord) toContest() Contest {
	divisions := make([]Division, len(c.Divisions))
	for i, d := range c.Divisions {
		divisions[i] = d.Division
	}
	return Contest{ID: c.ID, Name: c.Name, OwnerUserID: c.OwnerUserID, Divisions: divisions}
}

// Store is an in-memory, mutex-protected backing store for the ScoringApi.
// It mirrors frontend/src/api/mock.ts's shape and seed data so the same
// demo login/contest works against either implementation.
type Store struct {
	mu       sync.Mutex
	users    []User
	contests []*contestRecord
}

func NewStore() *Store {
	s := &Store{}
	s.seed()
	return s
}

func emptyRawScores(playerID string) PlayerRawScores {
	return PlayerRawScores{
		PlayerID:   playerID,
		Clickers:   map[int]ClickerInput{},
		Deductions: MajorDeductions{},
		Evals:      map[int]map[string]float64{},
	}
}

// clickerJudgeNames and evalJudgeNames mirror
// frontend/src/api/mock.ts's CLICKER_JUDGE_NAMES/EVAL_JUDGE_NAMES.
var clickerJudgeNames = [6][2]string{
	{"Agus", "Setiawan"}, {"Bambang", "Wijaya"}, {"Candra", "Saputra"},
	{"Doni", "Firmansyah"}, {"Eko", "Prasetyo"}, {"Fajar", "Nugroho"},
}
var evalJudgeNames = [6][2]string{
	{"Gita", "Lestari"}, {"Hendra", "Gunawan"}, {"Indah", "Puspitasari"},
	{"Joko", "Susanto"}, {"Kartika", "Dewi"}, {"Lestari", "Wulandari"},
}

func (s *Store) seed() {
	headJudge := User{ID: newID("u"), FirstName: "Galih", LastName: "Kurniawan", Email: "galih@example.com"}
	var clickerJudges, evalJudges []User
	for i := 0; i < 6; i++ {
		clickerJudges = append(clickerJudges, User{
			ID: newID("u"), FirstName: clickerJudgeNames[i][0], LastName: clickerJudgeNames[i][1],
			Email: fmt.Sprintf("%s.a%d@example.com", strings.ToLower(clickerJudgeNames[i][0]), i+1),
		})
		evalJudges = append(evalJudges, User{
			ID: newID("u"), FirstName: evalJudgeNames[i][0], LastName: evalJudgeNames[i][1],
			Email: fmt.Sprintf("%s.b%d@example.com", strings.ToLower(evalJudgeNames[i][0]), i+1),
		})
	}
	s.users = append([]User{headJudge}, append(append([]User{}, clickerJudges...), evalJudges...)...)

	players := []Player{
		{ID: newID("p"), Number: 1, Name: "Taro Yamada"},
		{ID: newID("p"), Number: 2, Name: "Jane Smith"},
		{ID: newID("p"), Number: 3, Name: "Budi Santoso"},
		{ID: newID("p"), Number: 4, Name: "Mei Lin"},
	}
	divisionID := newID("d")
	for i := range players {
		players[i].DivisionID = divisionID
	}

	contestID := newID("c")
	var assignments []JudgeAssignment
	for _, stage := range []calc.ScoringStage{calc.StagePrelim, calc.StageFinal} {
		for i, u := range clickerJudges {
			assignments = append(assignments, JudgeAssignment{
				ID: newID("a"), ContestID: contestID, DivisionID: divisionID,
				Stage: stage, UserID: u.ID, Role: RoleClicker, Slot: i + 1,
			})
		}
		for i, u := range evalJudges {
			assignments = append(assignments, JudgeAssignment{
				ID: newID("a"), ContestID: contestID, DivisionID: divisionID,
				Stage: stage, UserID: u.ID, Role: RoleEvaluator, Slot: i + 1,
			})
		}
	}

	finalNets := [][6]int{
		{50, 52, 51, 40, 48, 53},
		{55, 53, 54, 45, 50, 55},
		{40, 41, 39, 35, 38, 42},
		{58, 57, 59, 50, 56, 58},
	}
	finalScores := make([]PlayerRawScores, len(players))
	for i, p := range players {
		raw := emptyRawScores(p.ID)
		for j := 0; j < 6; j++ {
			raw.Clickers[j+1] = ClickerInput{Plus: finalNets[i][j], Minus: 0}
		}
		for j := 0; j < 6; j++ {
			raw.Evals[j+1] = map[string]float64{
				"EXE": 4 + float64(i%2)*0.5,
				"CTL": 4,
				"TDV": 3.5 + float64(j%2)*0.5,
				"SEM": 4,
				"MU1": 4,
				"MU2": 3.5,
				"BDY": 4,
				"SHW": 4 - float64(i%2)*0.5,
			}
		}
		if i == 2 {
			raw.Deductions = MajorDeductions{Stop: 1}
		}
		finalScores[i] = raw
	}

	prelimScores := make([]PlayerRawScores, len(players))
	for i, p := range players {
		prelimScores[i] = emptyRawScores(p.ID)
	}

	division := &divisionRecord{
		Division: Division{
			ID: divisionID, ContestID: contestID, Name: "3A",
			Stages: []calc.ScoringStage{calc.StagePrelim, calc.StageFinal},
		},
		Players:     players,
		Assignments: assignments,
		Scores: map[calc.ScoringStage][]PlayerRawScores{
			calc.StageFinal:  finalScores,
			calc.StagePrelim: prelimScores,
		},
	}

	s.contests = []*contestRecord{{
		ID: contestID, Name: "Indonesia National Yoyo Championships", OwnerUserID: headJudge.ID,
		Divisions: []*divisionRecord{division},
	}}
}

func (s *Store) findUserByEmail(email string) (User, bool) {
	for _, u := range s.users {
		if u.Email == email {
			return u, true
		}
	}
	return User{}, false
}

func (s *Store) findUserByID(id string) (User, bool) {
	for _, u := range s.users {
		if u.ID == id {
			return u, true
		}
	}
	return User{}, false
}

func (s *Store) findDivision(divisionID string) (*contestRecord, *divisionRecord, bool) {
	for _, c := range s.contests {
		for _, d := range c.Divisions {
			if d.ID == divisionID {
				return c, d, true
			}
		}
	}
	return nil, nil, false
}

func (s *Store) findContest(contestID string) (*contestRecord, bool) {
	for _, c := range s.contests {
		if c.ID == contestID {
			return c, true
		}
	}
	return nil, false
}

func (d *divisionRecord) rawScoresFor(stage calc.ScoringStage, playerID string) *PlayerRawScores {
	if d.Scores == nil {
		d.Scores = map[calc.ScoringStage][]PlayerRawScores{}
	}
	list := d.Scores[stage]
	for i := range list {
		if list[i].PlayerID == playerID {
			return &list[i]
		}
	}
	list = append(list, emptyRawScores(playerID))
	d.Scores[stage] = list
	return &list[len(list)-1]
}

func toPlayerInput(d *divisionRecord, stage calc.ScoringStage, p Player) calc.PlayerInput {
	raw := d.rawScoresFor(stage, p.ID)

	clickers := make([]calc.ClickerScore, 6)
	for i := 0; i < 6; i++ {
		if c, ok := raw.Clickers[i+1]; ok {
			clickers[i] = calc.ClickerScore{Plus: c.Plus, Minus: c.Minus}
		}
	}
	evalScores := make([]map[string]float64, 6)
	for i := 0; i < 6; i++ {
		if e, ok := raw.Evals[i+1]; ok {
			evalScores[i] = e
		} else {
			evalScores[i] = map[string]float64{}
		}
	}

	return calc.PlayerInput{
		Number:     p.Number,
		Name:       p.Name,
		Clickers:   [6]calc.ClickerScore(clickers),
		EvalScores: [6]map[string]float64(evalScores),
		Deductions: map[string]int{
			"Stop":    raw.Deductions.Stop,
			"Discard": raw.Deductions.Discard,
			"Cut":     raw.Deductions.Cut,
			"Detach":  raw.Deductions.Cut,
		},
	}
}
