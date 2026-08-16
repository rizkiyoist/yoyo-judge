// In-memory + localStorage-backed implementation of ScoringApi, seeded with
// demo data. Stands in for the real Go HTTP API until it exists.
import { calculate, newContest, type PlayerInput } from '../lib/scoring'
import type {
  ClickerInput,
  Contest,
  Division,
  JudgeAssignment,
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
  ownerUserId: string
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

function seedDb(): Db {
  const headJudge: User = { id: uid('u'), firstName: 'Alice', lastName: 'Head', email: 'alice@example.com' }
  const clickerJudges: User[] = Array.from({ length: 6 }, (_, i) => ({
    id: uid('u'),
    firstName: 'Clicker',
    lastName: `Judge A${i + 1}`,
    email: `clicker.a${i + 1}@example.com`,
  }))
  const evalJudges: User[] = Array.from({ length: 6 }, (_, i) => ({
    id: uid('u'),
    firstName: 'Eval',
    lastName: `Judge B${i + 1}`,
    email: `eval.b${i + 1}@example.com`,
  }))
  const users = [headJudge, ...clickerJudges, ...evalJudges]

  const players: Player[] = [
    { id: uid('p'), divisionId: '', number: 1, name: 'Taro Yamada' },
    { id: uid('p'), divisionId: '', number: 2, name: 'Jane Smith' },
    { id: uid('p'), divisionId: '', number: 3, name: 'Budi Santoso' },
    { id: uid('p'), divisionId: '', number: 4, name: 'Mei Lin' },
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
    name: 'WYYC Demo Contest',
    ownerUserId: headJudge.id,
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
    ownerUserId: dc.ownerUserId,
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

    async listContests(userId) {
      const owned = db.contests.filter((c) => c.ownerUserId === userId)
      const invited = db.contests.filter(
        (c) => c.ownerUserId !== userId && c.divisions.some((d) => d.assignments.some((a) => a.userId === userId)),
      )
      return delay([...owned, ...invited].map(toContest))
    },

    async getContest(contestId) {
      const contest = db.contests.find((c) => c.id === contestId)
      return delay(contest ? toContest(contest) : null)
    },

    async createContest(name, ownerUserId) {
      const contest: DbContest = { id: uid('c'), name, ownerUserId, divisions: [] }
      db.contests.push(contest)
      saveDb(db)
      return delay(toContest(contest))
    },

    async addDivision(contestId, name, stages) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) throw new Error('contest not found')
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
      division.stages = stages
      saveDb(db)
      return delay({ id: division.id, contestId, name: division.name, stages })
    },

    async listJudgeAssignments(contestId) {
      const contest = db.contests.find((c) => c.id === contestId)
      if (!contest) return delay([])
      return delay(contest.divisions.flatMap((d) => d.assignments))
    },

    async inviteJudge(contestId, divisionId, stage, userId, role, slot) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      found.division.assignments = found.division.assignments.filter(
        (a) => !(a.stage === stage && a.role === role && a.slot === slot),
      )
      const assignment: JudgeAssignment = { id: uid('a'), contestId, divisionId, stage, userId, role, slot }
      found.division.assignments.push(assignment)
      saveDb(db)
      return delay(assignment)
    },

    async removeJudgeAssignment(_contestId, assignmentId) {
      for (const contest of db.contests) {
        for (const division of contest.divisions) {
          division.assignments = division.assignments.filter((a) => a.id !== assignmentId)
        }
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
      const player: Player = { id: uid('p'), divisionId, number, name }
      found.division.players.push(player)
      saveDb(db)
      return delay(player)
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
      const raw = rawScoresFor(found.division, stage, playerId)
      raw.clickers[slot] = score
      saveDb(db)
      return delay(undefined)
    },

    async submitDeductions(divisionId, stage, playerId, deductions) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
      const raw = rawScoresFor(found.division, stage, playerId)
      raw.deductions = deductions
      saveDb(db)
      return delay(undefined)
    },

    async submitEvalScore(divisionId, stage, playerId, slot, scores) {
      const found = findDivision(db, divisionId)
      if (!found) throw new Error('division not found')
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
