/*
 * Created on Sun Aug 16 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"yoyo-judge/library/calc"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func readJSON(r *http.Request, dst any) bool {
	return json.NewDecoder(r.Body).Decode(dst) == nil
}

// nonNil turns a nil slice into an empty one before JSON encoding — Go
// marshals nil slices as `null`, which breaks frontend code expecting an array.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// Mount registers every ScoringApi HTTP route under {basePath}/api.
func Mount(router *mux.Router, store *Store, basePath string) {
	api := router.PathPrefix(basePath + "/api").Subrouter()

	// Auth — Google OAuth endpoints are GET (browser redirect flow).
	api.HandleFunc("/auth/login", store.handleLogin).Methods(http.MethodPost)
	api.HandleFunc("/auth/me", store.handleMe).Methods(http.MethodGet)
	api.HandleFunc("/auth/logout", store.handleLogout).Methods(http.MethodPost)
	api.HandleFunc("/auth/google", store.handleGoogleLogin).Methods(http.MethodGet)
	api.HandleFunc("/auth/google/callback", store.handleGoogleCallback).Methods(http.MethodGet)
	// Public: login screen calls this before a session exists to list demo users.
	api.HandleFunc("/users/search", store.handleSearchUsers).Methods(http.MethodGet)

	api.HandleFunc("/contests", store.requireAuth(store.handleListContests)).Methods(http.MethodGet)
	api.HandleFunc("/contests", store.requireAuth(store.handleCreateContest)).Methods(http.MethodPost)
	api.HandleFunc("/contests/{contestId}", store.requireAuth(store.handleGetContest)).Methods(http.MethodGet)
	api.HandleFunc("/contests/{contestId}/divisions", store.requireAuth(store.handleAddDivision)).Methods(http.MethodPost)
	api.HandleFunc("/contests/{contestId}/divisions/{divisionId}", store.requireAuth(store.handleUpdateDivisionStages)).Methods(http.MethodPatch)
	api.HandleFunc("/contests/{contestId}/divisions/{divisionId}", store.requireAuth(store.handleDeleteDivision)).Methods(http.MethodDelete)
	api.HandleFunc("/contests/{contestId}/lock", store.requireAuth(store.handleSetContestLock)).Methods(http.MethodPatch)
	api.HandleFunc("/contests/{contestId}/head-judge", store.requireAuth(store.handleTransferHeadJudge)).Methods(http.MethodPatch)

	api.HandleFunc("/contests/{contestId}/judges", store.requireAuth(store.handleListJudgeAssignments)).Methods(http.MethodGet)
	api.HandleFunc("/contests/{contestId}/judges", store.requireAuth(store.handleInviteJudge)).Methods(http.MethodPost)
	api.HandleFunc("/contests/{contestId}/judges/{assignmentId}", store.requireAuth(store.handleRemoveJudgeAssignment)).Methods(http.MethodDelete)

	api.HandleFunc("/divisions/{divisionId}/players", store.requireAuth(store.handleListPlayers)).Methods(http.MethodGet)
	api.HandleFunc("/divisions/{divisionId}/players", store.requireAuth(store.handleAddPlayer)).Methods(http.MethodPost)
	api.HandleFunc("/divisions/{divisionId}/players/{playerId}", store.requireAuth(store.handleRemovePlayer)).Methods(http.MethodDelete)

	api.HandleFunc("/divisions/{divisionId}/scores", store.requireAuth(store.handleGetRawScores)).Methods(http.MethodGet)
	api.HandleFunc("/divisions/{divisionId}/scores/clicker", store.requireAuth(store.handleSubmitClickerScore)).Methods(http.MethodPost)
	api.HandleFunc("/divisions/{divisionId}/scores/deductions", store.requireAuth(store.handleSubmitDeductions)).Methods(http.MethodPost)
	api.HandleFunc("/divisions/{divisionId}/scores/eval", store.requireAuth(store.handleSubmitEvalScore)).Methods(http.MethodPost)

	api.HandleFunc("/divisions/{divisionId}/results", store.requireAuth(store.handleGetResults)).Methods(http.MethodGet)
}

// --- auth ---

func (s *Store) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	user, ok := s.findUserByEmail(body.Email)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	token := s.createSession(user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "token": token})
}

func (s *Store) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := s.bearerUser(r)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (s *Store) handleLogout(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(header, "Bearer "); ok && token != "" {
		s.deleteSession(token)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	if idsParam := strings.TrimSpace(r.URL.Query().Get("ids")); idsParam != "" {
		writeJSON(w, http.StatusOK, nonNil(s.usersByIDs(strings.Split(idsParam, ","))))
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, nonNil(s.searchUsers(q)))
}

// --- contests ---

func (s *Store) handleListContests(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, nonNil(s.listAllContests()))
}

func (s *Store) handleCreateContest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Year int    `json:"year"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	writeJSON(w, http.StatusCreated, s.createContest(body.Name, body.Year, currentUser(r).ID))
}

func (s *Store) handleGetContest(w http.ResponseWriter, r *http.Request) {
	c, ok := s.getContest(mux.Vars(r)["contestId"])
	if !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Store) handleAddDivision(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	if status, msg := s.authorizeContestWrite(contestID, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	var body struct {
		Name   string              `json:"name"`
		Stages []calc.ScoringStage `json:"stages"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if _, ok := s.getContest(contestID); !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	writeJSON(w, http.StatusCreated, s.addDivision(contestID, body.Name, body.Stages))
}

func (s *Store) handleUpdateDivisionStages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if status, msg := s.authorizeContestWrite(vars["contestId"], currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	var body struct {
		Stages []calc.ScoringStage `json:"stages"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	division, ok := s.updateDivisionStages(vars["contestId"], vars["divisionId"], body.Stages)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	writeJSON(w, http.StatusOK, division)
}

// handleDeleteDivision removes a division. Refuses (409) if it still has
// players — see store.go's deleteDivision.
func (s *Store) handleDeleteDivision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if status, msg := s.authorizeContestWrite(vars["contestId"], currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	if status, msg := s.deleteDivision(vars["contestId"], vars["divisionId"]); status != 0 {
		writeError(w, status, msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSetContestLock lets only the current head judge freeze/unfreeze
// the entire contest. Deliberately does NOT go through
// authorizeContestWrite's lock check — locking/unlocking must work
// regardless of the contest's current locked state, or a locked contest
// could never be unlocked.
func (s *Store) handleSetContestLock(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	var body struct {
		Locked bool `json:"locked"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !s.isHeadJudge(contestID, currentUser(r).ID) {
		writeError(w, http.StatusForbidden, "only the head judge can lock or unlock this contest")
		return
	}
	contest, ok := s.setContestLocked(contestID, body.Locked)
	if !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	writeJSON(w, http.StatusOK, contest)
}

// handleTransferHeadJudge lets the owner or current head judge hand
// head-judge privileges to another judge already invited to the contest
// (or back to the owner).
func (s *Store) handleTransferHeadJudge(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	if status, msg := s.authorizeContestWrite(contestID, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	var body struct {
		UserID string `json:"userId"`
	}
	if !readJSON(r, &body) || body.UserID == "" {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	contest, status, msg := s.transferHeadJudge(contestID, body.UserID)
	if status != 0 {
		writeError(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, contest)
}

// --- judges ---

func (s *Store) handleListJudgeAssignments(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	if _, ok := s.getContest(contestID); !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	writeJSON(w, http.StatusOK, nonNil(s.listJudgeAssignments(contestID)))
}

func (s *Store) handleInviteJudge(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	if status, msg := s.authorizeContestWrite(contestID, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	var body struct {
		DivisionID string            `json:"divisionId"`
		Stage      calc.ScoringStage `json:"stage"`
		UserID     string            `json:"userId"`
		Email      string            `json:"email"`
		Role       JudgeRole         `json:"role"`
		Slot       int               `json:"slot"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	userID := body.UserID
	if userID == "" {
		email := strings.TrimSpace(body.Email)
		if email == "" {
			writeError(w, http.StatusBadRequest, "userId or email is required")
			return
		}
		user, err := s.findOrCreateUserByEmail(email)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create user: "+err.Error())
			return
		}
		userID = user.ID
	}
	writeJSON(w, http.StatusCreated, s.inviteJudge(contestID, body.DivisionID, body.Stage, userID, body.Role, body.Slot))
}

func (s *Store) handleRemoveJudgeAssignment(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	if status, msg := s.authorizeContestWrite(contestID, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	if !s.removeJudgeAssignment(contestID, mux.Vars(r)["assignmentId"]) {
		writeError(w, http.StatusNotFound, "judge assignment not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- players ---

func (s *Store) handleListPlayers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, nonNil(s.listPlayers(mux.Vars(r)["divisionId"])))
}

func (s *Store) handleAddPlayer(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	contestID, ok := s.divisionContestID(divisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	if status, msg := s.authorizeContestWrite(contestID, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	var body struct {
		Number int    `json:"number"`
		Name   string `json:"name"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	writeJSON(w, http.StatusCreated, s.addPlayer(divisionID, body.Number, body.Name))
}

func (s *Store) handleRemovePlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	divisionID := vars["divisionId"]
	contestID, ok := s.divisionContestID(divisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	if status, msg := s.authorizeContestWrite(contestID, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	if !s.removePlayer(divisionID, vars["playerId"]) {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- scoring ---

func (s *Store) handleGetRawScores(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	stage := calc.ScoringStage(r.URL.Query().Get("stage"))
	writeJSON(w, http.StatusOK, s.getRawScoresForDivision(divisionID, stage))
}

func (s *Store) handleSubmitClickerScore(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	var body struct {
		Stage    calc.ScoringStage `json:"stage"`
		PlayerID string            `json:"playerId"`
		Slot     int               `json:"slot"`
		Score    ClickerInput      `json:"score"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if status, msg := s.authorizeSlotScoreWrite(divisionID, body.Stage, RoleClicker, body.Slot, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	s.upsertClickerScore(divisionID, body.Stage, body.PlayerID, body.Slot, body.Score)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) handleSubmitDeductions(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	var body struct {
		Stage      calc.ScoringStage `json:"stage"`
		PlayerID   string            `json:"playerId"`
		Deductions MajorDeductions   `json:"deductions"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if status, msg := s.authorizeDeductionsWrite(divisionID, body.Stage, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	s.upsertDeductions(divisionID, body.Stage, body.PlayerID, body.Deductions)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) handleSubmitEvalScore(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	var body struct {
		Stage    calc.ScoringStage  `json:"stage"`
		PlayerID string             `json:"playerId"`
		Slot     int                `json:"slot"`
		Scores   map[string]float64 `json:"scores"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if status, msg := s.authorizeSlotScoreWrite(divisionID, body.Stage, RoleEvaluator, body.Slot, currentUser(r).ID); status != 0 {
		writeError(w, status, msg)
		return
	}
	s.upsertEvalScore(divisionID, body.Stage, body.PlayerID, body.Slot, body.Scores)
	w.WriteHeader(http.StatusNoContent)
}

// --- results ---

func (s *Store) handleGetResults(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	stage := calc.ScoringStage(r.URL.Query().Get("stage"))
	writeJSON(w, http.StatusOK, nonNil(s.calculateResults(divisionID, stage)))
}
