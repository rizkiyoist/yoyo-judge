/*
 * Created on Sun Aug 16 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package server

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

// bearerUser resolves the Authorization: Bearer <token> header to a User via
// the sessions table. Returns false when the token is missing or expired.
func (s *Store) bearerUser(r *http.Request) (User, bool) {
	header := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return User{}, false
	}
	return s.sessionUser(token)
}

// requireAuth rejects the request with 401 unless a valid bearer token is
// present, and stashes the resolved User in the request context.
func (s *Store) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := s.bearerUser(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid Authorization bearer token")
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func currentUser(r *http.Request) User {
	u, _ := r.Context().Value(userContextKey).(User)
	return u
}
