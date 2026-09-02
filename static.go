/*
 * Created on Fri Aug 21 2026
 *
 * Copyright (c) Rizki Hadiaturrasyid
 */

package main

import "embed"

// bin/static/ holds the built frontend (frontend/dist, copied here by
// build.ps1 before compiling) so the frontend ships embedded inside the
// backend binary — one binary, one port, no separate static host needed.
// This is the same copy build.ps1 also hands to anyone hosting the
// frontend separately (the cross-origin split deploy shape) — one build
// output, not two. It's gitignored (like the rest of bin/); run build.ps1
// (or at least its frontend step) once after a fresh clone before
// `go build .`/`go run .` will succeed, since //go:embed requires this
// directory to exist at compile time.
//
//go:embed all:bin/static
var staticFiles embed.FS
