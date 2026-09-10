import { ref, watchEffect } from 'vue'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'yoyo-judge-theme'

function systemPrefersDark(): boolean {
  return window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false
}

const stored = localStorage.getItem(STORAGE_KEY) as Theme | null
export const theme = ref<Theme>(stored ?? (systemPrefersDark() ? 'dark' : 'light'))

watchEffect(() => {
  document.documentElement.setAttribute('data-theme', theme.value)
  localStorage.setItem(STORAGE_KEY, theme.value)
})

export function toggleTheme(): void {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
}

export type ViewMode = 'auto' | 'mobile' | 'desktop'

const VIEW_MODE_KEY = 'yoyo-judge-view-mode'
const MOBILE_QUERY = '(max-width: 720px)'

function readViewMode(): ViewMode {
  const stored = localStorage.getItem(VIEW_MODE_KEY)
  return stored === 'mobile' || stored === 'desktop' ? stored : 'auto'
}

export const viewMode = ref<ViewMode>(readViewMode())

// Auto mode follows the viewport — mirror the existing @media (max-width:
// 720px) breakpoint so manual mobile and auto-on-a-narrow-screen render
// identically.
const viewportIsMobile = ref<boolean>(window.matchMedia?.(MOBILE_QUERY).matches ?? false)
window.matchMedia?.(MOBILE_QUERY).addEventListener('change', (e) => {
  viewportIsMobile.value = e.matches
})

watchEffect(() => {
  const resolvedMobile =
    viewMode.value === 'mobile' || (viewMode.value === 'auto' && viewportIsMobile.value)
  document.documentElement.toggleAttribute('data-mobile-view', resolvedMobile)
  localStorage.setItem(VIEW_MODE_KEY, viewMode.value)
})

const CYCLE: ViewMode[] = ['auto', 'mobile', 'desktop']
export function cycleViewMode(): void {
  const i = CYCLE.indexOf(viewMode.value)
  viewMode.value = CYCLE[(i + 1) % CYCLE.length]
}
