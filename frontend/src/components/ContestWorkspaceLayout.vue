<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useContestStore } from '../stores/contests'
import type { ScoringStage } from '../types'

const props = defineProps<{
  contestId: string
  activeTab: 'divisions' | 'judges' | 'players' | 'input-score' | 'result-detail'
}>()

const store = useContestStore()
const route = useRoute()

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
</script>

<template>
  <RouterLink class="workspace-crumb" :to="{ name: 'contests' }">← Back to contests</RouterLink>
  <nav class="tabs-bar">
    <div class="tabs-inner">
      <RouterLink class="tab" :class="{ active: activeTab === 'divisions' }" :to="{ name: 'contest-edit', params: { contestId } }">Divisions</RouterLink>
      <RouterLink class="tab" :class="{ active: activeTab === 'judges' }" :to="{ name: 'contest-judges', params: { contestId } }">Judges</RouterLink>
      <RouterLink v-if="playersTarget" class="tab" :class="{ active: activeTab === 'players' }" :to="playersTarget">Players</RouterLink>
      <span v-else class="tab" style="opacity: 0.4; cursor: not-allowed;" title="Add a division first">Players</span>
      <RouterLink v-if="inputScoreTarget" class="tab" :class="{ active: activeTab === 'input-score' }" :to="inputScoreTarget">Input Score</RouterLink>
      <span v-else class="tab" style="opacity: 0.4; cursor: not-allowed;" title="Add a division with a stage first">Input Score</span>
      <RouterLink v-if="resultDetailTarget" class="tab" :class="{ active: activeTab === 'result-detail' }" :to="resultDetailTarget">Result Detail</RouterLink>
      <span v-else class="tab" style="opacity: 0.4; cursor: not-allowed;" title="Add a division with a stage first">Result Detail</span>
    </div>
  </nav>
  <slot />
</template>
