<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useContestStore } from '../stores/contests'
import type { ScoringStage } from '../types'

const props = defineProps<{
  contestId: string
  activeTab: 'divisions' | 'judges' | 'players' | 'input-score' | 'result-detail'
}>()

const store = useContestStore()
const route = useRoute()
const router = useRouter()

onMounted(() => {
  if (!store.contests.length) store.fetchContests()
})

const contest = computed(() => store.contests.find((c) => c.id === props.contestId))

// Players / Input Score / Result Detail need a division (and, for the last
// two, a stage). If the current URL already carries those params, keep them
// as the tab target so the active tab click doesn't jump the user off the
// division/stage they're already viewing. Otherwise fall back to the
// contest's first division and its earliest stage.
const currentDivisionId = computed(() => (route.params.divisionId as string | undefined) ?? '')
const currentStage = computed(() => (route.params.stage as ScoringStage | undefined) ?? '')

function stageForDivision(divisionId: string): ScoringStage | null {
  const div = contest.value?.divisions.find((d) => d.id === divisionId)
  if (!div) return null
  if (div.stages.includes('prelim')) return 'prelim'
  if (div.stages.includes('final')) return 'final'
  return null
}

const playersTarget = computed(() => {
  const divisionId = currentDivisionId.value || contest.value?.divisions[0]?.id
  if (!divisionId) return null
  return { name: 'division-players', params: { contestId: props.contestId, divisionId } }
})
const inputScoreTarget = computed(() => {
  const divisionId = currentDivisionId.value || contest.value?.divisions[0]?.id
  if (!divisionId) return null
  const stage = currentStage.value || stageForDivision(divisionId)
  if (!stage) return null
  return { name: 'score-entry', params: { contestId: props.contestId, divisionId, stage } }
})
const resultDetailTarget = computed(() => {
  const divisionId = currentDivisionId.value || contest.value?.divisions[0]?.id
  if (!divisionId) return null
  const stage = currentStage.value || stageForDivision(divisionId)
  if (!stage) return null
  return { name: 'results', params: { contestId: props.contestId, divisionId, stage } }
})

// Single source of truth for the tab strip: the template renders this and
// the swipe gesture walks it, so the two can't fall out of order. A null
// `to` renders as a disabled tab and is skipped when swiping.
const tabs = computed(() => [
  { key: 'divisions', label: 'Divisions', to: { name: 'contest-edit', params: { contestId: props.contestId } }, hint: '' },
  { key: 'judges', label: 'Judges', to: { name: 'contest-judges', params: { contestId: props.contestId } }, hint: '' },
  { key: 'players', label: 'Players', to: playersTarget.value, hint: 'Add a division first' },
  { key: 'input-score', label: 'Input Score', to: inputScoreTarget.value, hint: 'Add a division with a stage first' },
  { key: 'result-detail', label: 'Result Detail', to: resultDetailTarget.value, hint: 'Add a division with a stage first' },
])

// --- Swipe between tabs (touch only) ---
// Left = next tab, right = previous. No wrap-around: swiping past either
// end does nothing, so you can't loop from Result Detail back to Divisions.
const SWIPE_MIN_PX = 60
// Horizontal travel must beat vertical by this much, otherwise a diagonal
// flick during normal vertical scrolling would fire a tab change.
const SWIPE_DOMINANCE = 1.5

let startX = 0
let startY = 0
let tracking = false

// Score tables and the tab strip itself scroll sideways inside the page.
// A swipe that starts in one of those belongs to that element, not to us.
function startedInHorizontalScroller(target: EventTarget | null, root: Element): boolean {
  let node = target instanceof Element ? target : null
  while (node && node !== root) {
    if (node.scrollWidth > node.clientWidth + 1) {
      const overflowX = getComputedStyle(node).overflowX
      if (overflowX === 'auto' || overflowX === 'scroll') return true
    }
    node = node.parentElement
  }
  return false
}

function onTouchStart(e: TouchEvent): void {
  if (e.touches.length !== 1) {
    tracking = false
    return
  }
  startX = e.touches[0].clientX
  startY = e.touches[0].clientY
  tracking = !startedInHorizontalScroller(e.target, e.currentTarget as Element)
}

function onTouchEnd(e: TouchEvent): void {
  if (!tracking) return
  tracking = false
  const dx = e.changedTouches[0].clientX - startX
  const dy = e.changedTouches[0].clientY - startY
  if (Math.abs(dx) < SWIPE_MIN_PX) return
  if (Math.abs(dx) < Math.abs(dy) * SWIPE_DOMINANCE) return
  goToAdjacentTab(dx < 0 ? 1 : -1)
}

function goToAdjacentTab(step: number): void {
  const enabled = tabs.value.filter((t) => t.to)
  const i = enabled.findIndex((t) => t.key === props.activeTab)
  if (i === -1) return
  const next = enabled[i + step]
  if (next?.to) router.push(next.to)
}
</script>

<template>
  <RouterLink class="workspace-crumb" :to="{ name: 'contests' }">← Back to contests</RouterLink>
  <div class="workspace-swipe" @touchstart.passive="onTouchStart" @touchend.passive="onTouchEnd">
    <nav class="tabs-bar">
      <div class="tabs-inner">
        <template v-for="t in tabs" :key="t.key">
          <RouterLink v-if="t.to" class="tab" :class="{ active: activeTab === t.key }" :to="t.to">{{ t.label }}</RouterLink>
          <span v-else class="tab tab-disabled" :title="t.hint">{{ t.label }}</span>
        </template>
      </div>
    </nav>
    <slot />
  </div>
</template>
