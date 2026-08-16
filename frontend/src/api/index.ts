import { createHttpApi } from './http'
import { useMockApi } from './mock'

// Single seam for swapping ScoringApi implementations. Defaults to the real
// Go backend (server/); set VITE_USE_MOCK=true to fall back to the
// in-browser mock for offline frontend-only work.
export const api = import.meta.env.VITE_USE_MOCK === 'true' ? useMockApi() : createHttpApi()
export type { ScoringApi } from './client'
