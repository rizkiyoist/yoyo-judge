// API contract for the scoring backend. `mock.ts` implements this against an
// in-memory/localStorage store; swapping in the real Go HTTP API later means
// writing one more implementation of this interface, not touching any view.
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

export interface ScoringApi {
  // Auth (mocked: no real password check, just picks a seeded user by email)
  login(email: string): Promise<User | null>
  currentUser(): Promise<User | null>
  logout(): Promise<void>
  searchUsers(query: string): Promise<User[]>
  getUsers(ids: string[]): Promise<User[]>

  // Contests
  listContests(userId: string): Promise<Contest[]>
  getContest(contestId: string): Promise<Contest | null>
  createContest(name: string, year: number, ownerUserId: string): Promise<Contest>
  addDivision(contestId: string, name: string, stages: ScoringStage[]): Promise<Division>
  updateDivisionStages(contestId: string, divisionId: string, stages: ScoringStage[]): Promise<Division>

  // Judges
  listJudgeAssignments(contestId: string): Promise<JudgeAssignment[]>
  // `identity` is either an existing user's id, or an email to invite by —
  // if no user exists with that email yet, one is created with no name set
  // (an "unclaimed" placeholder), so the invite is already waiting for them
  // the first time they actually log in with that email/Google account.
  inviteJudge(
    contestId: string,
    divisionId: string,
    stage: ScoringStage,
    identity: { userId: string } | { email: string },
    role: JudgeRole,
    slot: number,
  ): Promise<JudgeAssignment>
  removeJudgeAssignment(contestId: string, assignmentId: string): Promise<void>

  // Players
  listPlayers(divisionId: string): Promise<Player[]>
  addPlayer(divisionId: string, number: number, name: string): Promise<Player>

  // Scoring
  getRawScores(divisionId: string, stage: ScoringStage): Promise<PlayerRawScores[]>
  submitClickerScore(
    divisionId: string,
    stage: ScoringStage,
    playerId: string,
    slot: number,
    score: ClickerInput,
  ): Promise<void>
  submitDeductions(
    divisionId: string,
    stage: ScoringStage,
    playerId: string,
    deductions: MajorDeductions,
  ): Promise<void>
  submitEvalScore(
    divisionId: string,
    stage: ScoringStage,
    playerId: string,
    slot: number,
    scores: Record<string, number>,
  ): Promise<void>

  // Results
  getResults(divisionId: string, stage: ScoringStage): Promise<PlayerResult[]>
}
