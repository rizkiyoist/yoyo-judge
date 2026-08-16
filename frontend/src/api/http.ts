// Real HTTP-backed ScoringApi implementation, calling the Go backend
// (server/) at VITE_API_BASE_URL (default http://localhost:5000/api).
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

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:5000/api'
const TOKEN_KEY = 'yoyo-judge-token'

function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

function setToken(token: string | null): void {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers = new Headers(init?.headers)
  headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? `Request to ${path} failed with ${res.status}`)
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

function query(params: Record<string, string | undefined>): string {
  const usable = Object.entries(params).filter(([, v]) => v !== undefined) as [string, string][]
  if (!usable.length) return ''
  return `?${new URLSearchParams(usable).toString()}`
}

export function createHttpApi(): ScoringApi {
  return {
    async login(email) {
      const { user, token } = await request<{ user: User | null; token?: string }>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email }),
      })
      setToken(user ? (token ?? null) : null)
      return user
    },

    async currentUser() {
      if (!getToken()) return null
      const { user } = await request<{ user: User | null }>('/auth/me')
      return user
    },

    async logout() {
      await request('/auth/logout', { method: 'POST' })
      setToken(null)
    },

    async searchUsers(q) {
      return request<User[]>(`/users/search${query({ q })}`)
    },

    async listContests(_userId) {
      return request<Contest[]>('/contests')
    },

    async getContest(contestId) {
      try {
        return await request<Contest>(`/contests/${contestId}`)
      } catch {
        return null
      }
    },

    async createContest(name, year, _ownerUserId) {
      return request<Contest>('/contests', { method: 'POST', body: JSON.stringify({ name, year }) })
    },

    async addDivision(contestId, name, stages: ScoringStage[]) {
      return request<Division>(`/contests/${contestId}/divisions`, {
        method: 'POST',
        body: JSON.stringify({ name, stages }),
      })
    },

    async updateDivisionStages(contestId, divisionId, stages: ScoringStage[]) {
      return request<Division>(`/contests/${contestId}/divisions/${divisionId}`, {
        method: 'PATCH',
        body: JSON.stringify({ stages }),
      })
    },

    async listJudgeAssignments(contestId) {
      return request<JudgeAssignment[]>(`/contests/${contestId}/judges`)
    },

    async inviteJudge(contestId, divisionId, stage, userId, role: JudgeRole, slot) {
      return request<JudgeAssignment>(`/contests/${contestId}/judges`, {
        method: 'POST',
        body: JSON.stringify({ divisionId, stage, userId, role, slot }),
      })
    },

    async removeJudgeAssignment(contestId, assignmentId) {
      await request(`/contests/${contestId}/judges/${assignmentId}`, { method: 'DELETE' })
    },

    async listPlayers(divisionId) {
      return request<Player[]>(`/divisions/${divisionId}/players`)
    },

    async addPlayer(divisionId, number, name) {
      return request<Player>(`/divisions/${divisionId}/players`, {
        method: 'POST',
        body: JSON.stringify({ number, name }),
      })
    },

    async getRawScores(divisionId, stage) {
      return request<PlayerRawScores[]>(`/divisions/${divisionId}/scores${query({ stage })}`)
    },

    async submitClickerScore(divisionId, stage, playerId, slot, score: ClickerInput) {
      await request(`/divisions/${divisionId}/scores/clicker`, {
        method: 'POST',
        body: JSON.stringify({ stage, playerId, slot, score }),
      })
    },

    async submitDeductions(divisionId, stage, playerId, deductions: MajorDeductions) {
      await request(`/divisions/${divisionId}/scores/deductions`, {
        method: 'POST',
        body: JSON.stringify({ stage, playerId, deductions }),
      })
    },

    async submitEvalScore(divisionId, stage, playerId, slot, scores) {
      await request(`/divisions/${divisionId}/scores/eval`, {
        method: 'POST',
        body: JSON.stringify({ stage, playerId, slot, scores }),
      })
    },

    async getResults(divisionId, stage) {
      return request<PlayerResult[]>(`/divisions/${divisionId}/results${query({ stage })}`)
    },
  }
}
