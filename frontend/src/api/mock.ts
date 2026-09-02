// In-memory + localStorage-backed implementation of ScoringApi, seeded with
// demo data. Stands in for the real Go HTTP API until it exists.
import { calculate, newContest, type PlayerInput } from '../lib/scoring'
import type {
  ClickerInput,
  Contest,
  Division,
  JudgeAssignment,
  JudgeRole,
  MajorDeductions,
  Player,
  PlayerRawScores,
  PlayerResult,
  ScoringStage,
  User,
} from '../types'
import type { ScoringApi } from './client'

interface DbDivision extends Division {
  players: Player[]
  assignments: JudgeAssignment[]
  scores: Partial<Record<ScoringStage, PlayerRawScores[]>>
}

interface DbContest {
  id: string
  name: string
  year: number
  ownerUserId: string
  headJudgeUserId: string
  locked: boolean
  divisions: DbDivision[]
}

interface Db {
  users: User[]
  sessionUserId: string | null
  contests: DbContest[]
}

const STORAGE_KEY = 'yoyo-judge-mock-db-v1'

function uid(prefix: string): string {
  return `${prefix}_${crypto.randomUUID().slice(0, 8)}`
}

function delay<T>(value: T, ms = 150): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms))
}

function emptyDeductions(): MajorDeductions {
  return { stop: 0, discard: 0, cut: 0 }
}

function emptyRawScores(playerId: string): PlayerRawScores {
  return { playerId, clickers: {}, deductions: emptyDeductions(), evals: {} }
}

const CLICKER_JUDGE_NAMES: [string, string][] = [
  ['Paketu', 'Dennis'],
  ['Levian', 'Saputra'],
  ['Boris', 'Chietra'],
  ['Doni', 'Firmansyah'],
  ['Eko', 'Prasetyo'],
  ['Fajar', 'Nugroho'],
]
const EVAL_JUDGE_NAMES: [string, string][] = [
  ['Reynold', 'Andika'],
  ['Hendra', 'Kusumah'],
  ['Indah', 'Puspitasari'],
  ['Joko', 'Susanto'],
  ['Kartika', 'Dewi'],
  ['Lestari', 'Wulandari'],
]

function seedDb(): Db {
  const headJudge: User = { id: uid('u'), firstName: 'Galih', lastName: 'Kurniawan', email: 'galih@example.com' }
  const clickerJudges: User[] = CLICKER_JUDGE_NAMES.map(([firstName, lastName], i) => ({
    id: uid('u'),
    firstName,
    lastName,
    email: `${firstName.toLowerCase()}.a${i + 1}@example.com`,
  }))
  const evalJudges: User[] = EVAL_JUDGE_NAMES.map(([firstName, lastName], i) => ({
    id: uid('u'),
    firstName,
    lastName,
    email: `${firstName.toLowerCase()}.b${i + 1}@example.com`,
  }))
  const users = [headJudge, ...clickerJudges, ...evalJudges]

  const players: Player[] = [
    { id: uid('p'), divisionId: '', number: 1, name: 'Taro Yamada' },
    { id: uid('p'), divisionId: '', number: 2, name: 'Jane Smith' },
    { id: uid('p'), divisionId: '', number: 3, name: 'Budi Santoso' },
    { id: uid('p'), divisionId: '', number: 4, name: 'Mei Lin' },
    { id: uid('p'), divisionId: '', number: 5, name: 'Wira Kusuma' },
    { id: uid('p'), divisionId: '', number: 6, name: 'Siti Aminah' },
    { id: uid('p'), divisionId: '', number: 7, name: 'Chen Wei' },
    { id: uid('p'), divisionId: '', number: 8, name: 'Aiko Tanaka' },
    { id: uid('p'), divisionId: '', number: 9, name: 'Diego Santos' },
    { id: uid('p'), divisionId: '', number: 10, name: 'Maria Garcia' },
  ]

  const divisionId = uid('d')
  players.forEach((p) => (p.divisionId = divisionId))

  const assignments: JudgeAssignment[] = []
  const stagesForDemo: ScoringStage[] = ['prelim', 'final']
  for (const stage of stagesForDemo) {
    clickerJudges.forEach((u, i) => {
      assignments.push({
        id: uid('a'),
        contestId: '',
        divisionId,
        stage,
        userId: u.id,
        role: 'clicker',
        slot: i + 1,
      })
    })
    evalJudges.forEach((u, i) => {
      assignments.push({
        id: uid('a'),
        contestId: '',
        divisionId,
        stage,
        userId: u.id,
        role: 'evaluator',
        slot: i + 1,
      })
    })
  }

  // Seed a handful of raw scores for the FINAL stage so the results screen
  // has something to show immediately.
  const finalNets = [
    [50, 52, 51, 40, 48, 53],
    [55, 53, 54, 45, 50, 55],
    [40, 41, 39, 35, 38, 42],
    [58, 57, 59, 50, 56, 58],
    [45, 44, 46, 42, 43, 45],
    [62, 61, 63, 60, 62, 64],
    [38, 37, 39, 36, 38, 40],
    [51, 50, 52, 49, 50, 53],
    [47, 46, 48, 45, 47, 49],
    [55, 54, 56, 53, 55, 57],
  ]
  const finalScores: PlayerRawScores[] = players.map((p, i) => {
    const raw = emptyRawScores(p.id)
    clickerJudges.forEach((_, j) => {
      raw.clickers[j + 1] = { plus: finalNets[i][j], minus: 0 }
    })
    evalJudges.forEach((_, j) => {
      raw.evals[j + 1] = {
        EXE: 4 + (i % 2) * 0.5,
        CTL: 4,
        TDV: 3.5 + (j % 2) * 0.5,
        SEM: 4,
        MU1: 4,
        MU2: 3.5,
        BDY: 4,
        SHW: 4 - (i % 2) * 0.5,
      }
    })
    raw.deductions = { stop: i === 2 ? 1 : 0, discard: 0, cut: 0 }
    return raw
  })

  const contestId = uid('c')
  assignments.forEach((a) => (a.contestId = contestId))

  const division: DbDivision = {
    id: divisionId,
    contestId,
    name: '3A',
    stages: ['prelim', 'final'],
    players,
    assignments,
    scores: { final: finalScores, prelim: players.map((p) => emptyRawScores(p.id)) },
  }

  const contest: DbContest = {
    id: contestId,
    name: 'Indonesia National Yoyo Championships',
    year: 2026,
    ownerUserId: headJudge.id,
    headJudgeUserId: headJudge.id,
    locked: false,
    divisions: [division],
  }

  return { users, sessionUserId: headJudge.id, contests: [contest] }
}

function loadDb(): Db {
  const raw = localStorage.getItem(STORAGE_KEY)
  if (!raw) {
    const db = seedDb()
    saveDb(db)
    return db
  }
  try {
    return JSON.parse(raw) as Db
  } catch {
    const db = seedDb()
    saveDb(db)
    return db
  }
}

function saveDb(db: Db): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(db))
}

function toContest(dc: DbContest): Contest {
  return {
    id: dc.id,
    name: dc.name,
    year: dc.year,
    ownerUserId: dc.ownerUserId,
    headJudgeUserId: dc.headJudgeUserId || dc.ownerUserId,
    locked: dc.locked,
    divisions: dc.divisions.map((d) => ({
      id: d.id,
      contestId: d.contestId,
      name: d.name,
      stages: d.stages,
    })),
  }
}

function findDivision(db: Db, divisionId: string): { contest: DbContest; division: DbDivision } | null {
  for (const contest of db.contests) {
    const division = contest.divisions.find((d) => d.id === divisionId)
    if (division) return { contest, division }
  }
  return null
}

// Mirrors server/store.go's role helpers: the owner never changes; the
// head judge defaults to the owner but is transferable (see
// transferHeadJudge below).
function isHeadJudge(contest: DbContest, userId: string | null): boolean {
  return !!userId && (contest.headJudgeUserId || contest.ownerUserId) === userId
}
function isOwnerOrHeadJudge(contest: DbContest, userId: string | null): boolean {
  return !!userId && (contest.ownerUserId === userId || isHeadJudge(contest, userId))
}

// Mirrors server/store.go's authorizeContestWrite: used for adding/
// removing divisions, players, and judges, and for transferring
// head-judge status.
function assertContestWritable(contest: DbContest, userId: string | null): void {
  if (contest.locked) throw new Error('this contest is locked')
  if (!isOwnerOrHeadJudge(contest, userId)) {
    throw new Error('only the contest owner or head judge can do this')
  }
}

// Mirrors server/store.go's authorizeSlotScoreWrite: a locked contest
// rejects everyone, even the head judge (who must unlock first); the head
// judge otherwise always may (the override page's "act as any judge"),
// anyone else must hold exactly this division+stage+role+slot.
function assertSlotScoreWritable(
  contest: DbContest,
  division: DbDivision,
  stage: ScoringStage,
  role: JudgeRole,
  slot: number,
  userId: string | null,
): void {
  if (contest.locked) throw new Error('this contest is locked')
  if (isHeadJudge(contest, userId)) return
  const assigned = division.assignments.some(
    (a) => a.stage === stage && a.role === role && a.slot === slot && a.userId === userId,
  )
  if (!assigned) throw new Error('you are not the assigned judge for this slot')
}

// Mirrors server/store.go's authorizeDeductionsWrite: any assigned judge
// (either role, not just one slot) may record deductions.
function assertDeductionsWritable(
  contest: DbContest,
  division: DbDivision,
  stage: ScoringStage,
  userId: string | null,
): void {
  if (contest.locked) throw new Error('this contest is locked')
  if (isHeadJudge(contest, userId)) return
  const assigned = division.assignments.some((a) => a.stage === stage && a.userId === userId)
  if (!assigned) throw new Error('you are not an assigned judge for this division/stage')
}

function rawScoresFor(division: DbDivision, stage: ScoringStage, playerId: string): PlayerRawScores {
  const list = (division.scores[stage] ??= division.players.map((p) => emptyRawScores(p.id)))
  let entry = list.find((s) => s.playerId === playerId)
  if (!entry) {
    entry = emptyRawScores(playerId)
    list.push(entry)
  }
  return entry
}

function toPlayerInput(division: DbDivision, stage: ScoringStage, player: Player): PlayerInput {
  const raw = rawScoresFor(division, stage, player.id)
  const clickerSlots = division.assignments.filter((a) => a.stage === stage && a.role === 'clicker')
  const evalSlots = division.assignments.filter((a) => a.stage === stage && a.role === 'evaluator')

  const clickers = Array.from({ length: 6 }, (_, i) => {
    const slot = clickerSlots.find((a) => a.slot === i + 1)?.slot ?? i + 1
    return raw.clickers[slot] ?? { plus: 0, minus: 0 }
  })
  const evalScores = Array.from({ length: 6 }, (_, i) => {
    const slot = evalSlots.find((a) => a.slot === i + 1)?.slot ?? i + 1
    return raw.evals[slot] ?? {}
  })

  return {
    number: player.number,
    name: player.name,
    clickers,
    evalScores,
    deductions: { Stop: raw.deductions.stop, Discard: raw.deductions.discard, Cut: raw.deductions.cut },
  }
}

export function createMockApi(): ScoringApi {
  const db = loadDb()

  return {
    async login(email) {
      const user = db.users.find((u) => u.email.toLowerCase() === email.trim().toLowerCase()) ?? null
      db.sessionUserId = user?.id ?? null
      saveDb(db)
      return delay(user)
    },

    async currentUser() {
      const user = db.users.find((u) => u.id === db.sessionUserId) ?? null
      return delay(user)
    },

    async logout() {
      db.sessionUserId = null
      saveDb(db)
      return delay(undefined)
    },

    async searchUsers(query) {
      const q = query.trim().toLowerCase()
      if (!q) return delay([])
      const matches = db.users.filter(
        (u) => `${u.firstName} ${u.lastName}`.toLowerCase().includes(q) || u.email.toLowerCase().includes(q),
      )
      return delay(matches)
    },

    async getUsers(ids) {
      const idSet = new Set(ids)
      return delay(db.users.filter((u) => idSet.has(u.id)))
    },

    async listContests() {
      return delay(db.contests.map(toContest))
    },

    async getContest(contestId) {
      const contest = db.contests.find((c) => c.id === contestId)
      return delay(contest ? toContest(contest) : null)
    },

    async createContest(name, year, ownerUserId) {
      const contest: DbContest = {
        id: uid('c'),
        name,
        year,
        ownerUserId,
        headJudgeUserId: ownerUserId,
        locked: false,
        divisions: [],
      }
      db.contests.push(contest)
      saveDb(db)
      return delay(toContest(contest))
    },

    async addDivision(contestId, name, stages) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) throw new Error('contest not found')
      assertContestWritable(contest, db.sessionUserId)
      const division: DbDivision = {
        id: uid('d'),
        contestId,
        name,
        stages,
        players: [],
        assignments: [],
        scores: {},
      }
      contest.divisions.push(division)
      saveDb(db)
      return delay({ id: division.id, contestId, name, stages })
    },

    async updateDivisionStages(contestId, divisionId, stages) {
      const contest = db.contests.find((c) => c.id === contestId)
      const division = contest?.divisions.find((d) => d.id === divisionId)
      if (!contest || !division) throw new Error('division not found')
      assertContestWritable(contest, db.sessionUserId)
      division.stages = stages
      saveDb(db)
      return delay({ id: division.id, contestId, name: division.name, stages })
    },

    async deleteDivision(contestId, divisionId) {
      const contest = db.contests.find((c) => c.id === contestId)
      const division = contest?.divisions.find((d) => d.id === divisionId)
      if (!contest || !division) throw new Error('division not found')
      assertContestWritable(contest, db.sessionUserId)
      if (division.players.length > 0) {
        throw new Error('this division still has players - remove them all before deleting the division')
      }
      contest.divisions = contest.divisions.filter((d) => d.id !== divisionId)
      saveDb(db)
      return delay(undefined)
    },

    async setContestLocked(contestId, locked) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) throw new Error('contest not found')
      if (!isHeadJudge(contest, db.sessionUserId)) {
        throw new Error('only the head judge can lock or unlock this contest')
      }
      contest.locked = locked
      saveDb(db)
      return delay(toContest(contest))
    },

    async transferHeadJudge(contestId, userId) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) throw new Error('contest not found')
      assertContestWritable(contest, db.sessionUserId)
      const eligible =
        contest.ownerUserId === userId ||
        contest.divisions.some((d) => d.assignments.some((a) => a.userId === userId))
      if (!eligible) {
        throw new Error('the new head judge must be the contest owner or a judge already invited to this contest')
      }
      contest.headJudgeUserId = userId
      saveDb(db)
      return delay(toContest(contest))
    },

    async listJudgeAssignments(contestId) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) return delay([])
      return delay(contest.divisions.flatMap((d) => d.assignments))
    },

    async inviteJudge(contestId, divisionId, stage, identity, role, slot) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      assertContestWritable(found.contest, db.sessionUserId)
      let userId: string
      if ('userId' in identity) {
        userId = identity.userId
      } else {
        const email = identity.email.trim().toLowerCase()
        const existing = db.users.find((u) => u.email.toLowerCase() === email)
        if (existing) {
          userId = existing.id
        } else {
          // Placeholder account, no name yet - this is what makes the
          // invite already visible the first time this email actually logs in.
          const placeholder: User = { id: uid('u'), firstName: '', lastName: '', email }
          db.users.push(placeholder)
          userId = placeholder.id
        }
      }
      found.division.assignments = found.division.assignments.filter(
        (a) => !(a.stage === stage && a.role === role && a.slot === slot),
      )
      const assignment: JudgeAssignment = { id: uid('a'), contestId, divisionId, stage, userId, role, slot }
      found.division.assignments.push(assignment)
      saveDb(db)
      return delay(assignment)
    },

    async removeJudgeAssignment(contestId, assignmentId) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) throw new Error('contest not found')
      assertContestWritable(contest, db.sessionUserId)
      for (const division of contest.divisions) {
        division.assignments = division.assignments.filter((a) => a.id !== assignmentId)
      }
      saveDb(db)
      return delay(undefined)
    },

    async listPlayers(divisionId) {
      const found = findDivision(db, divisionId)
      return delay(found ? found.division.players : [])
    },

    async addPlayer(divisionId, number, name) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      assertContestWritable(found.contest, db.sessionUserId)
      const player: Player = { id: uid('p'), divisionId, number, name }
      found.division.players.push(player)
      saveDb(db)
      return delay(player)
    },

    async removePlayer(divisionId, playerId) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      assertContestWritable(found.contest, db.sessionUserId)
      found.division.players = found.division.players.filter((p) => p.id !== playerId)
      for (const stage of Object.keys(found.division.scores) as ScoringStage[]) {
        const list = found.division.scores[stage]
        if (list) found.division.scores[stage] = list.filter((s) => s.playerId !== playerId)
      }
      saveDb(db)
      return delay(undefined)
    },

    async getRawScores(divisionId, stage) {
      const found = findDivision(db, divisionId)
      if (!found) return delay([])
      const scores = found.division.players.map((p) => rawScoresFor(found.division, stage, p.id))
      return delay(structuredClone(scores))
    },

    async submitClickerScore(divisionId, stage, playerId, slot, score: ClickerInput) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      assertSlotScoreWritable(found.contest, found.division, stage, 'clicker', slot, db.sessionUserId)
      const raw = rawScoresFor(found.division, stage, playerId)
      raw.clickers[slot] = score
      saveDb(db)
      return delay(undefined)
    },

    async submitDeductions(divisionId, stage, playerId, deductions) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      assertDeductionsWritable(found.contest, found.division, stage, db.sessionUserId)
      const raw = rawScoresFor(found.division, stage, playerId)
      raw.deductions = deductions
      saveDb(db)
      return delay(undefined)
    },

    async submitEvalScore(divisionId, stage, playerId, slot, scores) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      assertSlotScoreWritable(found.contest, found.division, stage, 'evaluator', slot, db.sessionUserId)
      const raw = rawScoresFor(found.division, stage, playerId)
      raw.evals[slot] = scores
      saveDb(db)
      return delay(undefined)
    },

    async getResults(divisionId, stage) {
      const found = findDivision(db, divisionId)
      if (!found) return delay([])
      const { division } = found
      const playerInputs = division.players.map((p) => toPlayerInput(division, stage, p))
      const contest = newContest(stage, playerInputs)
      const results = calculate(contest)
      const withIds: PlayerResult[] = results.map((r, i) => ({
        playerId: division.players[i].id,
        ...r,
      }))
      return delay(withIds)
    },
  }
}

let apiInstance: ScoringApi | null = null

export function useMockApi(): ScoringApi {
  if (!apiInstance) apiInstance = createMockApi()
  return apiInstance
}
