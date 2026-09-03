// Frontend domain types. Mirrors domain/model/user.go and library/calc where
// those exist; the contest/division/judge-assignment concepts below don't
// exist in the Go backend yet - this is the contract the real API should
// eventually implement.
import type { ScoringStage } from './lib/scoring'

export type { ScoringStage }

export interface User {
  id: string
  firstName: string
  lastName: string
  email: string
  // Computed server-side from env.json's/SUPERADMIN_EMAILS' allowlist - a
  // superadmin may edit any contest's name/year and hide/show contests,
  // regardless of ownership.
  isSuperAdmin: boolean
}

export interface Division {
  id: string
  contestId: string
  name: string
  stages: ScoringStage[]
}

export interface Contest {
  id: string
  name: string
  year: number
  // Permanent creator - never changes. Distinct from headJudgeUserId,
  // which the owner (or the current head judge) can transfer to any
  // invited judge; the owner defaults to being the head judge too.
  ownerUserId: string
  headJudgeUserId: string
  // Freezes the whole contest (all divisions/scores/players/judges)
  // against writes, including by the head judge - only unlocking is
  // exempt. Toggled by the current head judge only.
  locked: boolean
  // Excludes this contest from a non-superadmin's contest list. Only a
  // superadmin can toggle it, and only a superadmin's listing ever
  // includes a hidden contest, so they can find it again to un-hide it.
  hidden: boolean
  divisions: Division[]
}

// 'major_deduction' is always an additional role on top of an existing
// clicker/evaluator assignment for the same judge - never assigned alone.
export type JudgeRole = 'clicker' | 'evaluator' | 'major_deduction'

export interface JudgeAssignment {
  id: string
  contestId: string
  divisionId: string
  stage: ScoringStage
  userId: string
  role: JudgeRole
  slot: number // 1-6, matches SET-UP J-A1..A6 / J-B1..B6
}

export interface Player {
  id: string
  divisionId: string
  number: number
  name: string
}

export interface ClickerInput {
  plus: number
  minus: number
}

export interface MajorDeductions {
  stop: number
  discard: number
  cut: number // "Detach" in the PRELIM stage, same field
}

// One player's raw judge inputs for one division+stage, as submitted so far
// by whichever judges have entered scores.
export interface PlayerRawScores {
  playerId: string
  clickers: Partial<Record<number, ClickerInput>> // slot -> score
  deductions: MajorDeductions
  evals: Partial<Record<number, Record<string, number>>> // slot -> category -> score
}

export interface PlayerResult {
  playerId: string
  number: number
  name: string
  technicalExecution: number
  categoryScores: Record<string, number>
  groupTotals: Partial<Record<'TEv' | 'PEv', number>>
  evaluationTotal: number
  deductionTotals: Record<string, number>
  deductionTotal: number
  finalScore: number
  place: number
}
