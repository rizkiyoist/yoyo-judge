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
	"net/http"
	"slices"
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
		updates := map[string]any{}
		if existing.GoogleID == "" {
			updates["google_id"] = googleID
		}
		// Backfill the name for a placeholder user created by an email
		// invite (findOrCreateUserByEmail leaves both blank) — never
		// overwrite a name that's already set.
		if existing.FirstName == "" && existing.LastName == "" {
			updates["first_name"] = firstName
			updates["last_name"] = lastName
		}
		if len(updates) > 0 {
			s.db.Model(&existing).Updates(updates)
			existing.GoogleID = googleID
			if _, ok := updates["first_name"]; ok {
				existing.FirstName, existing.LastName = firstName, lastName
			}
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

// findOrCreateUserByEmail is used when inviting a judge by email who hasn't
// signed in yet: it creates a placeholder user (no name) so the invite is
// already waiting for them — findOrCreateUserByGoogle matches this same row
// by email on their first real login (Google or demo) instead of creating a
// duplicate.
func (s *Store) findOrCreateUserByEmail(email string) (User, error) {
	var existing DBUser
	if s.db.Where("LOWER(email) = LOWER(?)", email).First(&existing).Error == nil {
		return dbUserToUser(existing), nil
	}
	u := DBUser{ID: newID("u"), Email: email}
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

// usersByIDs resolves a batch of user ids at once — used to look up the
// name/email of every judge referenced by a contest's assignments,
// including ones invited by email who haven't signed in yet.
func (s *Store) usersByIDs(ids []string) []User {
	if len(ids) == 0 {
		return []User{}
	}
	var rows []DBUser
	s.db.Where("id IN ?", ids).Find(&rows)
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

// listAllContests returns every contest — any signed-in user (any judge,
// not just the ones a contest has invited) can see every contest, though
// only its owner and its invited judges can change its scores (enforced
// separately, see isContestOwner/isAssignedSlot/isAssignedRole below).
func (s *Store) listAllContests() []Contest {
	var all []DBContest
	s.db.Order("created_at").Find(&all)
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
	var locked []calc.ScoringStage
	_ = json.Unmarshal([]byte(d.LockedStages), &locked)
	return Division{ID: d.ID, ContestID: d.ContestID, Name: d.Name, Stages: stages, LockedStages: nonNil(locked)}
}

func (s *Store) addDivision(contestID, name string, stages []calc.ScoringStage) Division {
	b, _ := json.Marshal(stages)
	d := DBDivision{ID: newID("d"), ContestID: contestID, Name: name, Stages: string(b), LockedStages: "[]"}
	s.db.Create(&d)
	return dbDivisionToDivision(d)
}

// setDivisionStageLock freezes (or unfreezes) one stage of a division
// against edits from anyone but the contest owner — see isStageLocked.
// Returns false if the division doesn't belong to contestID.
func (s *Store) setDivisionStageLock(contestID, divisionID string, stage calc.ScoringStage, locked bool) (Division, bool) {
	var d DBDivision
	if s.db.Where("id = ? AND contest_id = ?", divisionID, contestID).First(&d).Error != nil {
		return Division{}, false
	}
	var lockedStages []calc.ScoringStage
	_ = json.Unmarshal([]byte(d.LockedStages), &lockedStages)
	next := lockedStages[:0]
	found := false
	for _, s := range lockedStages {
		if s == stage {
			found = true
			if locked {
				next = append(next, s)
			}
			continue
		}
		next = append(next, s)
	}
	if locked && !found {
		next = append(next, stage)
	}
	b, _ := json.Marshal(next)
	s.db.Model(&d).Update("locked_stages", string(b))
	d.LockedStages = string(b)
	return dbDivisionToDivision(d), true
}

// isStageLocked reports whether a division's stage is currently frozen.
func (s *Store) isStageLocked(divisionID string, stage calc.ScoringStage) bool {
	var d DBDivision
	if s.db.First(&d, "id = ?", divisionID).Error != nil {
		return false
	}
	var lockedStages []calc.ScoringStage
	_ = json.Unmarshal([]byte(d.LockedStages), &lockedStages)
	return slices.Contains(lockedStages, stage)
}

// divisionContestID resolves a division to its contest id, for
// authorization checks on division-scoped routes that don't otherwise
// carry the contest id in the URL.
func (s *Store) divisionContestID(divisionID string) (string, bool) {
	var d DBDivision
	if s.db.Select("contest_id").First(&d, "id = ?", divisionID).Error != nil {
		return "", false
	}
	return d.ContestID, true
}

// isContestOwner reports whether userID created contestID.
func (s *Store) isContestOwner(contestID, userID string) bool {
	var c DBContest
	if s.db.Select("owner_user_id").First(&c, "id = ?", contestID).Error != nil {
		return false
	}
	return c.OwnerUserID == userID
}

// isAssignedSlot reports whether userID is the judge assigned to exactly
// this division+stage+role+slot — used to authorize a clicker/eval score
// submission for that specific slot.
func (s *Store) isAssignedSlot(divisionID string, stage calc.ScoringStage, role JudgeRole, slot int, userID string) bool {
	var count int64
	s.db.Model(&DBJudgeAssignment{}).Where(
		"division_id = ? AND stage = ? AND role = ? AND slot = ? AND user_id = ?",
		divisionID, string(stage), string(role), slot, userID,
	).Count(&count)
	return count > 0
}

// isAssignedRole reports whether userID holds any slot of the given role
// for this division+stage — used to authorize deductions, which any
// clicker judge (not just one specific slot) may submit.
func (s *Store) isAssignedRole(divisionID string, stage calc.ScoringStage, role JudgeRole, userID string) bool {
	var count int64
	s.db.Model(&DBJudgeAssignment{}).Where(
		"division_id = ? AND stage = ? AND role = ? AND user_id = ?",
		divisionID, string(stage), string(role), userID,
	).Count(&count)
	return count > 0
}

// authorizeSlotScoreWrite checks whether userID may submit a clicker/eval
// score for exactly this division+stage+role+slot: the contest owner
// always may (that's how the head-judge override page works, and locking
// is specifically meant to stop *other* judges, not them); anyone else
// must be the judge assigned to that slot, and only while the stage isn't
// locked. Returns status 0 (proceed) or an HTTP status + message to reject
// with.
func (s *Store) authorizeSlotScoreWrite(divisionID string, stage calc.ScoringStage, role JudgeRole, slot int, userID string) (status int, message string) {
	contestID, found := s.divisionContestID(divisionID)
	if !found {
		return http.StatusNotFound, "division not found"
	}
	if s.isContestOwner(contestID, userID) {
		return 0, ""
	}
	if s.isStageLocked(divisionID, stage) {
		return http.StatusLocked, "scores are locked by the head judge"
	}
	if !s.isAssignedSlot(divisionID, stage, role, slot, userID) {
		return http.StatusForbidden, "you are not the assigned judge for this slot"
	}
	return 0, ""
}

// authorizeDeductionsWrite is authorizeSlotScoreWrite's counterpart for
// major deductions, which any clicker judge for the division+stage (not
// just one specific slot) is allowed to record.
func (s *Store) authorizeDeductionsWrite(divisionID string, stage calc.ScoringStage, userID string) (status int, message string) {
	contestID, found := s.divisionContestID(divisionID)
	if !found {
		return http.StatusNotFound, "division not found"
	}
	if s.isContestOwner(contestID, userID) {
		return 0, ""
	}
	if s.isStageLocked(divisionID, stage) {
		return http.StatusLocked, "scores are locked by the head judge"
	}
	if !s.isAssignedRole(divisionID, stage, RoleClicker, userID) {
		return http.StatusForbidden, "you are not a clicker judge for this division/stage"
	}
	return 0, ""
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

// removeJudgeAssignment deletes an assignment, scoped to contestID so a
// contest owner can't delete another contest's assignment by pairing their
// own (owned) contestId in the URL with someone else's assignmentId.
// Reports whether a row was actually deleted.
func (s *Store) removeJudgeAssignment(contestID, assignmentID string) bool {
	res := s.db.Where("id = ? AND contest_id = ?", assignmentID, contestID).Delete(&DBJudgeAssignment{})
	return res.RowsAffected > 0
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
