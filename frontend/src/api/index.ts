import { useMockApi } from './mock'

// Single seam to swap in a real HTTP-backed ScoringApi later.
export const api = useMockApi()
export type { ScoringApi } from './client'
