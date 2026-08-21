/*
 * Created on Fri Aug 21 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package main

import "embed"

// dist/ holds the built frontend (frontend/dist, copied here by build.ps1
// before compiling) so the frontend ships embedded inside the backend
// binary — one binary, one port, no separate static host needed. It's
// gitignored (like bin/); run build.ps1 (or at least its frontend step)
// once after a fresh clone before `go build .`/`go run .` will succeed,
// since //go:embed requires this directory to exist at compile time.
//
//go:embed all:dist
var staticFiles embed.FS
