/*
 * Created on Sun Aug 16 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"yoyo-judge/library/calc"
)

var ErrNotFound = errors.New("not found")

func newID(prefix string) string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf))
}

// Store is the SQLite-backed implementation of the ScoringApi contract.
// All writes go through GORM; SQLite's WAL mode handles concurrent reads.
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// --- users ---

func (s *Store) findUserByEmail(email string) (User, bool) {
	var u DBUser
	if s.db.Where("email = ?", email).First(&u).Error != nil {
		return User{}, false
	}
	return dbUserToUser(u), true
}

func (s *Store) findUserByID(id string) (User, bool) {
	var u DBUser
	if s.db.First(&u, "id = ?", id).Error != nil {
		return User{}, false
	}
	return dbUserToUser(u), true
}

func (s *Store) findUserByGoogleID(googleID string) (User, bool) {
	if googleID == "" {
		return User{}, false
	}
	var u DBUser
	if s.db.Where("google_id = ?", googleID).First(&u).Error != nil {
		return User{}, false
	}
	return dbUserToUser(u), true
}

// findOrCreateUserByGoogle looks up a user by Google ID, falls back to email
// (linking the Google ID if not already set), and creates a new user if neither
// matches.
func (s *Store) findOrCreateUserByGoogle(googleID, email, firstName, lastName string) (User, error) {
	if u, ok := s.findUserByGoogleID(googleID); ok {
		return u, nil
	}
	var existing DBUser
	if s.db.Where("email = ?", email).First(&existing).Error == nil {
		if existing.GoogleID == "" {
			s.db.Model(&existing).Update("google_id", googleID)
		}
		return dbUserToUser(existing), nil
	}
	u := DBUser{
		ID: newID("u"), FirstName: firstName, LastName: lastName,
		Email: email, GoogleID: googleID,
	}
	if err := s.db.Create(&u).Error; err != nil {
		return User{}, err
	}
	return dbUserToUser(u), nil
}

func (s *Store) searchUsers(q string) []User {
	if q == "" {
		return []User{}
	}
	var rows []DBUser
	like := "%" + q + "%"
	s.db.Where(
		"LOWER(first_name || ' ' || last_name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)",
		like, like,
	).Find(&rows)
	result := make([]User, len(rows))
	for i, u := range rows {
		result[i] = dbUserToUser(u)
	}
	return result
}

func dbUserToUser(u DBUser) User {
	return User{ID: u.ID, FirstName: u.FirstName, LastName: u.LastName, Email: u.Email}
}

// --- sessions ---

func (s *Store) createSession(userID string) string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := hex.EncodeToString(buf)
	s.db.Create(&DBSession{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	return token
}

func (s *Store) sessionUser(token string) (User, bool) {
	var sess DBSession
	if s.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&sess).Error != nil {
		return User{}, false
	}
	return s.findUserByID(sess.UserID)
}

func (s *Store) deleteSession(token string) {
	s.db.Where("token = ?", token).Delete(&DBSession{})
}

// --- contests ---

func (s *Store) listContestsForUser(userID string) []Contest {
	var owned []DBContest
	s.db.Where("owner_user_id = ?", userID).Find(&owned)

	var judgeContestIDs []string
	s.db.Model(&DBJudgeAssignment{}).Where("user_id = ?", userID).
		Distinct("contest_id").Pluck("contest_id", &judgeContestIDs)
	ownedSet := make(map[string]bool, len(owned))
	for _, c := range owned {
		ownedSet[c.ID] = true
	}
	var inviteIDs []string
	for _, id := range judgeContestIDs {
		if !ownedSet[id] {
			inviteIDs = append(inviteIDs, id)
		}
	}
	var invited []DBContest
	if len(inviteIDs) > 0 {
		s.db.Where("id IN ?", inviteIDs).Find(&invited)
	}

	all := append(owned, invited...)
	result := make([]Contest, len(all))
	for i, c := range all {
		result[i] = s.dbContestToContest(c)
	}
	return result
}

func (s *Store) dbContestToContest(c DBContest) Contest {
	var divs []DBDivision
	s.db.Where("contest_id = ?", c.ID).Find(&divs)
	divisions := make([]Division, len(divs))
	for i, d := range divs {
		divisions[i] = dbDivisionToDivision(d)
	}
	return Contest{ID: c.ID, Name: c.Name, Year: c.Year, OwnerUserID: c.OwnerUserID, Divisions: divisions}
}

func (s *Store) getContest(contestID string) (Contest, bool) {
	var c DBContest
	if s.db.First(&c, "id = ?", contestID).Error != nil {
		return Contest{}, false
	}
	return s.dbContestToContest(c), true
}

func (s *Store) createContest(name string, year int, ownerUserID string) Contest {
	c := DBContest{ID: newID("c"), Name: name, Year: year, OwnerUserID: ownerUserID}
	s.db.Create(&c)
	return Contest{ID: c.ID, Name: c.Name, Year: c.Year, OwnerUserID: c.OwnerUserID, Divisions: []Division{}}
}

// --- divisions ---

func dbDivisionToDivision(d DBDivision) Division {
	var stages []calc.ScoringStage
	_ = json.Unmarshal([]byte(d.Stages), &stages)
	return Division{ID: d.ID, ContestID: d.ContestID, Name: d.Name, Stages: stages}
}

func (s *Store) addDivision(contestID, name string, stages []calc.ScoringStage) Division {
	b, _ := json.Marshal(stages)
	d := DBDivision{ID: newID("d"), ContestID: contestID, Name: name, Stages: string(b)}
	s.db.Create(&d)
	return dbDivisionToDivision(d)
}

func (s *Store) updateDivisionStages(contestID, divisionID string, stages []calc.ScoringStage) (Division, bool) {
	b, _ := json.Marshal(stages)
	res := s.db.Model(&DBDivision{}).
		Where("id = ? AND contest_id = ?", divisionID, contestID).
		Update("stages", string(b))
	if res.RowsAffected == 0 {
		return Division{}, false
	}
	var d DBDivision
	s.db.First(&d, "id = ?", divisionID)
	return dbDivisionToDivision(d), true
}

// --- judge assignments ---

func (s *Store) listJudgeAssignments(contestID string) []JudgeAssignment {
	var rows []DBJudgeAssignment
	s.db.Where("contest_id = ?", contestID).Find(&rows)
	result := make([]JudgeAssignment, len(rows))
	for i, a := range rows {
		result[i] = dbAssignmentToAssignment(a)
	}
	return result
}

func dbAssignmentToAssignment(a DBJudgeAssignment) JudgeAssignment {
	return JudgeAssignment{
		ID: a.ID, ContestID: a.ContestID, DivisionID: a.DivisionID,
		Stage: calc.ScoringStage(a.Stage), UserID: a.UserID,
		Role: JudgeRole(a.Role), Slot: a.Slot,
	}
}

func (s *Store) inviteJudge(contestID, divisionID string, stage calc.ScoringStage, userID string, role JudgeRole, slot int) JudgeAssignment {
	// Replace any existing assignment for the same stage/role/slot.
	s.db.Where("division_id = ? AND stage = ? AND role = ? AND slot = ?",
		divisionID, string(stage), string(role), slot).Delete(&DBJudgeAssignment{})
	a := DBJudgeAssignment{
		ID: newID("a"), ContestID: contestID, DivisionID: divisionID,
		Stage: string(stage), UserID: userID, Role: string(role), Slot: slot,
	}
	s.db.Create(&a)
	return dbAssignmentToAssignment(a)
}

func (s *Store) removeJudgeAssignment(assignmentID string) {
	s.db.Delete(&DBJudgeAssignment{}, "id = ?", assignmentID)
}

// --- players ---

func (s *Store) listPlayers(divisionID string) []Player {
	var rows []DBPlayer
	s.db.Where("division_id = ?", divisionID).Order("number").Find(&rows)
	result := make([]Player, len(rows))
	for i, p := range rows {
		result[i] = Player{ID: p.ID, DivisionID: p.DivisionID, Number: p.Number, Name: p.Name}
	}
	return result
}

func (s *Store) addPlayer(divisionID string, number int, name string) Player {
	p := DBPlayer{ID: newID("p"), DivisionID: divisionID, Number: number, Name: name}
	s.db.Create(&p)
	return Player{ID: p.ID, DivisionID: p.DivisionID, Number: p.Number, Name: p.Name}
}

// --- raw scores ---

func (s *Store) getOrCreateRawScore(divisionID, playerID string, stage calc.ScoringStage) DBPlayerRawScore {
	var row DBPlayerRawScore
	err := s.db.Where("division_id = ? AND player_id = ? AND stage = ?",
		divisionID, playerID, string(stage)).First(&row).Error
	if err != nil {
		row = DBPlayerRawScore{
			DivisionID: divisionID, PlayerID: playerID, Stage: string(stage),
			Clickers: "{}", Deductions: "{}", Evals: "{}",
		}
		s.db.Create(&row)
	}
	return row
}

func (s *Store) upsertClickerScore(divisionID string, stage calc.ScoringStage, playerID string, slot int, score ClickerInput) {
	row := s.getOrCreateRawScore(divisionID, playerID, stage)
	var clickers map[int]ClickerInput
	_ = json.Unmarshal([]byte(row.Clickers), &clickers)
	if clickers == nil {
		clickers = map[int]ClickerInput{}
	}
	clickers[slot] = score
	b, _ := json.Marshal(clickers)
	s.db.Model(&row).Update("clickers", string(b))
}

func (s *Store) upsertDeductions(divisionID string, stage calc.ScoringStage, playerID string, deductions MajorDeductions) {
	row := s.getOrCreateRawScore(divisionID, playerID, stage)
	b, _ := json.Marshal(deductions)
	s.db.Model(&row).Update("deductions", string(b))
}

func (s *Store) upsertEvalScore(divisionID string, stage calc.ScoringStage, playerID string, slot int, scores map[string]float64) {
	row := s.getOrCreateRawScore(divisionID, playerID, stage)
	var evals map[int]map[string]float64
	_ = json.Unmarshal([]byte(row.Evals), &evals)
	if evals == nil {
		evals = map[int]map[string]float64{}
	}
	evals[slot] = scores
	b, _ := json.Marshal(evals)
	s.db.Model(&row).Update("evals", string(b))
}

func (s *Store) getRawScoresForDivision(divisionID string, stage calc.ScoringStage) []PlayerRawScores {
	players := s.listPlayers(divisionID)
	result := make([]PlayerRawScores, len(players))
	for i, p := range players {
		row := s.getOrCreateRawScore(divisionID, p.ID, stage)
		result[i] = dbRowToRawScores(row)
	}
	return result
}

func dbRowToRawScores(row DBPlayerRawScore) PlayerRawScores {
	var clickers map[int]ClickerInput
	_ = json.Unmarshal([]byte(row.Clickers), &clickers)
	if clickers == nil {
		clickers = map[int]ClickerInput{}
	}
	var deductions MajorDeductions
	_ = json.Unmarshal([]byte(row.Deductions), &deductions)
	var evals map[int]map[string]float64
	_ = json.Unmarshal([]byte(row.Evals), &evals)
	if evals == nil {
		evals = map[int]map[string]float64{}
	}
	return PlayerRawScores{PlayerID: row.PlayerID, Clickers: clickers, Deductions: deductions, Evals: evals}
}

func (s *Store) calculateResults(divisionID string, stage calc.ScoringStage) []PlayerResultResponse {
	players := s.listPlayers(divisionID)
	if len(players) == 0 {
		return []PlayerResultResponse{}
	}
	inputs := make([]calc.PlayerInput, len(players))
	for i, p := range players {
		row := s.getOrCreateRawScore(divisionID, p.ID, stage)
		inputs[i] = rawScoresToPlayerInput(p, dbRowToRawScores(row))
	}
	results := calc.NewContest(stage, inputs).Calculate()
	responses := make([]PlayerResultResponse, len(results))
	for i, res := range results {
		responses[i] = toPlayerResultResponse(players[i].ID, res)
	}
	return responses
}

func rawScoresToPlayerInput(p Player, raw PlayerRawScores) calc.PlayerInput {
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
