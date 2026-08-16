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

// nonNil turns a nil slice into an empty one before it's JSON-encoded — Go
// marshals a nil slice as `null`, which crashes frontend code expecting an
// array (e.g. `.filter`/`.length` on the response).
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// Mount registers every ScoringApi HTTP route (frontend/src/api/client.ts)
// under /api on the given router.
func Mount(router *mux.Router, store *Store) {
	api := router.PathPrefix("/api").Subrouter()

	api.HandleFunc("/auth/login", store.handleLogin).Methods(http.MethodPost)
	api.HandleFunc("/auth/me", store.handleMe).Methods(http.MethodGet)
	api.HandleFunc("/auth/logout", store.handleLogout).Methods(http.MethodPost)
	// Public (no auth): the login screen lists seeded users to pick from
	// before a session exists, mirroring mock.ts's searchUsers.
	api.HandleFunc("/users/search", store.handleSearchUsers).Methods(http.MethodGet)

	api.HandleFunc("/contests", store.requireAuth(store.handleListContests)).Methods(http.MethodGet)
	api.HandleFunc("/contests", store.requireAuth(store.handleCreateContest)).Methods(http.MethodPost)
	api.HandleFunc("/contests/{contestId}", store.requireAuth(store.handleGetContest)).Methods(http.MethodGet)
	api.HandleFunc("/contests/{contestId}/divisions", store.requireAuth(store.handleAddDivision)).Methods(http.MethodPost)
	api.HandleFunc("/contests/{contestId}/divisions/{divisionId}", store.requireAuth(store.handleUpdateDivisionStages)).Methods(http.MethodPatch)

	api.HandleFunc("/contests/{contestId}/judges", store.requireAuth(store.handleListJudgeAssignments)).Methods(http.MethodGet)
	api.HandleFunc("/contests/{contestId}/judges", store.requireAuth(store.handleInviteJudge)).Methods(http.MethodPost)
	api.HandleFunc("/contests/{contestId}/judges/{assignmentId}", store.requireAuth(store.handleRemoveJudgeAssignment)).Methods(http.MethodDelete)

	api.HandleFunc("/divisions/{divisionId}/players", store.requireAuth(store.handleListPlayers)).Methods(http.MethodGet)
	api.HandleFunc("/divisions/{divisionId}/players", store.requireAuth(store.handleAddPlayer)).Methods(http.MethodPost)

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
	s.mu.Lock()
	user, ok := s.findUserByEmail(body.Email)
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"user": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "token": user.ID})
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
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	s.mu.Lock()
	defer s.mu.Unlock()
	var matches []User
	if q != "" {
		for _, u := range s.users {
			name := strings.ToLower(u.FirstName + " " + u.LastName)
			email := strings.ToLower(u.Email)
			if strings.Contains(name, q) || strings.Contains(email, q) {
				matches = append(matches, u)
			}
		}
	}
	writeJSON(w, http.StatusOK, nonNil(matches))
}

// --- contests ---

func (s *Store) handleListContests(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	s.mu.Lock()
	defer s.mu.Unlock()

	var owned, invited []Contest
	for _, c := range s.contests {
		if c.OwnerUserID == user.ID {
			owned = append(owned, c.toContest())
			continue
		}
		for _, d := range c.Divisions {
			found := false
			for _, a := range d.Assignments {
				if a.UserID == user.ID {
					found = true
					break
				}
			}
			if found {
				invited = append(invited, c.toContest())
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, nonNil(append(owned, invited...)))
}

func (s *Store) handleCreateContest(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var body struct {
		Name string `json:"name"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	contest := &contestRecord{ID: newID("c"), Name: body.Name, OwnerUserID: user.ID}
	s.contests = append(s.contests, contest)
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, contest.toContest())
}

func (s *Store) handleGetContest(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.findContest(contestID)
	if !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	writeJSON(w, http.StatusOK, c.toContest())
}

func (s *Store) handleAddDivision(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	var body struct {
		Name   string              `json:"name"`
		Stages []calc.ScoringStage `json:"stages"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.findContest(contestID)
	if !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	division := &divisionRecord{
		Division: Division{ID: newID("d"), ContestID: contestID, Name: body.Name, Stages: body.Stages},
		Scores:   map[calc.ScoringStage][]PlayerRawScores{},
	}
	c.Divisions = append(c.Divisions, division)
	writeJSON(w, http.StatusCreated, division.Division)
}

func (s *Store) handleUpdateDivisionStages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var body struct {
		Stages []calc.ScoringStage `json:"stages"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(vars["divisionId"])
	if !ok || d.ContestID != vars["contestId"] {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	d.Stages = body.Stages
	writeJSON(w, http.StatusOK, d.Division)
}

// --- judges ---

func (s *Store) handleListJudgeAssignments(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.findContest(contestID)
	if !ok {
		writeError(w, http.StatusNotFound, "contest not found")
		return
	}
	var assignments []JudgeAssignment
	for _, d := range c.Divisions {
		assignments = append(assignments, d.Assignments...)
	}
	writeJSON(w, http.StatusOK, nonNil(assignments))
}

func (s *Store) handleInviteJudge(w http.ResponseWriter, r *http.Request) {
	contestID := mux.Vars(r)["contestId"]
	var body struct {
		DivisionID string            `json:"divisionId"`
		Stage      calc.ScoringStage `json:"stage"`
		UserID     string            `json:"userId"`
		Role       JudgeRole         `json:"role"`
		Slot       int               `json:"slot"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(body.DivisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	filtered := d.Assignments[:0]
	for _, a := range d.Assignments {
		if !(a.Stage == body.Stage && a.Role == body.Role && a.Slot == body.Slot) {
			filtered = append(filtered, a)
		}
	}
	assignment := JudgeAssignment{
		ID: newID("a"), ContestID: contestID, DivisionID: body.DivisionID,
		Stage: body.Stage, UserID: body.UserID, Role: body.Role, Slot: body.Slot,
	}
	d.Assignments = append(filtered, assignment)
	writeJSON(w, http.StatusCreated, assignment)
}

func (s *Store) handleRemoveJudgeAssignment(w http.ResponseWriter, r *http.Request) {
	assignmentID := mux.Vars(r)["assignmentId"]
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.contests {
		for _, d := range c.Divisions {
			filtered := d.Assignments[:0]
			for _, a := range d.Assignments {
				if a.ID != assignmentID {
					filtered = append(filtered, a)
				}
			}
			d.Assignments = filtered
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- players ---

func (s *Store) handleListPlayers(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeJSON(w, http.StatusOK, []Player{})
		return
	}
	writeJSON(w, http.StatusOK, nonNil(d.Players))
}

func (s *Store) handleAddPlayer(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	var body struct {
		Number int    `json:"number"`
		Name   string `json:"name"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	player := Player{ID: newID("p"), DivisionID: divisionID, Number: body.Number, Name: body.Name}
	d.Players = append(d.Players, player)
	writeJSON(w, http.StatusCreated, player)
}

// --- scoring ---

func (s *Store) handleGetRawScores(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	stage := calc.ScoringStage(r.URL.Query().Get("stage"))
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeJSON(w, http.StatusOK, []PlayerRawScores{})
		return
	}
	scores := make([]PlayerRawScores, 0, len(d.Players))
	for _, p := range d.Players {
		scores = append(scores, *d.rawScoresFor(stage, p.ID))
	}
	writeJSON(w, http.StatusOK, scores)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	raw := d.rawScoresFor(body.Stage, body.PlayerID)
	raw.Clickers[body.Slot] = body.Score
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
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	raw := d.rawScoresFor(body.Stage, body.PlayerID)
	raw.Deductions = body.Deductions
	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) handleSubmitEvalScore(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	var body struct {
		Stage    calc.ScoringStage `json:"stage"`
		PlayerID string            `json:"playerId"`
		Slot     int               `json:"slot"`
		Scores   map[string]float64 `json:"scores"`
	}
	if !readJSON(r, &body) {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeError(w, http.StatusNotFound, "division not found")
		return
	}
	raw := d.rawScoresFor(body.Stage, body.PlayerID)
	raw.Evals[body.Slot] = body.Scores
	w.WriteHeader(http.StatusNoContent)
}

// --- results ---

func (s *Store) handleGetResults(w http.ResponseWriter, r *http.Request) {
	divisionID := mux.Vars(r)["divisionId"]
	stage := calc.ScoringStage(r.URL.Query().Get("stage"))
	s.mu.Lock()
	defer s.mu.Unlock()
	_, d, ok := s.findDivision(divisionID)
	if !ok {
		writeJSON(w, http.StatusOK, []PlayerResultResponse{})
		return
	}

	inputs := make([]calc.PlayerInput, len(d.Players))
	for i, p := range d.Players {
		inputs[i] = toPlayerInput(d, stage, p)
	}
	contest := calc.NewContest(stage, inputs)
	results := contest.Calculate()

	responses := make([]PlayerResultResponse, len(results))
	for i, res := range results {
		responses[i] = toPlayerResultResponse(d.Players[i].ID, res)
	}
	writeJSON(w, http.StatusOK, responses)
}
