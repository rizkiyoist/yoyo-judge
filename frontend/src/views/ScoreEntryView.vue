<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { finalCategories, prelimCategories } from '../lib/scoring'
import { useAuthStore } from '../stores/auth'
import type { Contest, JudgeAssignment, Player, PlayerRawScores, ScoringStage } from '../types'

const props = defineProps<{ contestId: string; divisionId: string; stage: ScoringStage }>()

const auth = useAuthStore()
const contest = ref<Contest | null>(null)
const players = ref<Player[]>([])
const assignments = ref<JudgeAssignment[]>([])
const rawScores = ref<PlayerRawScores[]>([])
const DEFAULT_EVAL_SCORE = 5
// One page-wide save indicator rather than one per field: 'saving' while
// any edit is debouncing or in flight, 'saved' once everything settles.
type SaveState = 'idle' | 'saving' | 'saved'
const saveState = ref<SaveState>('idle')

// Debounce each field independently (keyed by section:playerId:field) so
// rapid clicks on a spinner only trigger one save, DEBOUNCE_MS after the
// user stops clicking - not one request per click.
const DEBOUNCE_MS = 2000
const pendingTimers: Record<string, ReturnType<typeof setTimeout>> = {}
let activeOps = 0
function statusKey(section: string, playerId: string, field: string): string {
  return `${section}:${playerId}:${field}`
}
function debounceSave(key: string, fn: () => Promise<void>) {
  if (pendingTimers[key]) {
    clearTimeout(pendingTimers[key])
  } else {
    activeOps++
  }
  saveState.value = 'saving'
  pendingTimers[key] = setTimeout(async () => {
    delete pendingTimers[key]
    try {
      await fn()
    } finally {
      activeOps--
      if (activeOps === 0) saveState.value = 'saved'
    }
  }, DEBOUNCE_MS)
}

const division = computed(() => contest.value?.divisions.find((d) => d.id === props.divisionId))
const isHeadJudge = computed(() => contest.value?.headJudgeUserId === auth.user?.id)
// Locking freezes the whole contest for everyone, including the head
// judge themself - they have to unlock (from the Judges page) before
// anyone, themself included, can submit or override scores again.
const inputsDisabled = computed(() => contest.value?.locked ?? false)

const myAssignments = computed(() =>
  assignments.value.filter(
    (a) => a.divisionId === props.divisionId && a.stage === props.stage && a.userId === auth.user?.id,
  ),
)
const myClickerAssignment = computed(() => myAssignments.value.find((a) => a.role === 'clicker'))
const myEvalAssignment = computed(() => myAssignments.value.find((a) => a.role === 'evaluator'))
// Only the one dedicated major-deduction judge for this division+stage may
// record deductions - always an additional role on top of clicker/eval.
// If nobody's been explicitly assigned yet, the head judge can still enter
// them here directly (the backend already lets the head judge through
// unconditionally) rather than only via the override page - covers the
// common case of forgetting to assign one.
const noMdAssigned = computed(
  () => !assignments.value.some((a) => a.divisionId === props.divisionId && a.stage === props.stage && a.role === 'major_deduction'),
)
const canEditDeductions = computed(
  () => myAssignments.value.some((a) => a.role === 'major_deduction') || (isHeadJudge.value && noMdAssigned.value),
)

const categories = computed(() => (props.stage === 'prelim' ? prelimCategories() : finalCategories()))
const thirdDeductionLabel = computed(() => (props.stage === 'prelim' ? 'Detach' : 'Cut'))

async function load() {
  contest.value = await api.getContest(props.contestId)
  players.value = await api.listPlayers(props.divisionId)
  assignments.value = await api.listJudgeAssignments(props.contestId)
  rawScores.value = await api.getRawScores(props.divisionId, props.stage)
}

onMounted(load)

function rawFor(playerId: string): PlayerRawScores | undefined {
  return rawScores.value.find((r) => r.playerId === playerId)
}

function saveClicker(playerId: string, field: 'plus' | 'minus', value: number) {
  if (!myClickerAssignment.value || inputsDisabled.value) return
  const key = statusKey('clicker', playerId, field)
  debounceSave(key, async () => {
    const slot = myClickerAssignment.value!.slot
    const existing = rawFor(playerId)?.clickers[slot] ?? { plus: 0, minus: 0 }
    const next = { ...existing, [field]: value }
    await api.submitClickerScore(props.divisionId, props.stage, playerId, slot, next)
    await load()
  })
}

function saveDeduction(playerId: string, field: 'stop' | 'discard' | 'cut', value: number) {
  if (inputsDisabled.value || !canEditDeductions.value) return
  const key = statusKey('deduction', playerId, field)
  debounceSave(key, async () => {
    const existing = rawFor(playerId)?.deductions ?? { stop: 0, discard: 0, cut: 0 }
    const next = { ...existing, [field]: value }
    await api.submitDeductions(props.divisionId, props.stage, playerId, next)
    await load()
  })
}

function saveEval(playerId: string, category: string, value: number) {
  if (!myEvalAssignment.value || inputsDisabled.value) return
  const key = statusKey('eval', playerId, category)
  debounceSave(key, async () => {
    const slot = myEvalAssignment.value!.slot
    const existing = rawFor(playerId)?.evals[slot] ?? {}
    const next = { ...existing, [category]: value }
    await api.submitEvalScore(props.divisionId, props.stage, playerId, slot, next)
    await load()
  })
}
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <h1>{{ contest.name }} - {{ division?.name }} ({{ stage }}) scoring</h1>

    <p v-if="isHeadJudge" class="row" style="align-items: center">
      <RouterLink :to="{ name: 'score-override', params: { contestId, divisionId, stage } }">
        <button>Head judge: override any judge's score</button>
      </RouterLink>
    </p>

    <p v-if="contest.locked" class="error">
      🔒 This contest is locked - go to the Judges page to unlock it before any scores can change.
    </p>

    <p
      v-if="saveState !== 'idle'"
      class="save-status"
      :class="saveState === 'saved' ? 'save-status--saved' : 'save-status--pending'"
    >
      {{ saveState === 'saved' ? 'All changes saved' : 'Saving…' }}
    </p>

    <p v-if="!myAssignments.length" class="muted">
      You aren't assigned as a judge for this division/stage. Ask the head judge to invite you from the Judges page.
    </p>

    <div v-if="myClickerAssignment" class="card">
      <h2>Clicker judge - slot {{ myClickerAssignment.slot }}</h2>
      <fieldset :disabled="inputsDisabled" style="border: 0; padding: 0; margin: 0">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Player</th>
            <th>+</th>
            <th>-</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in players" :key="p.id">
            <td>{{ p.number }}</td>
            <td>{{ p.name }}</td>
            <td>
              <input
                type="number" min="0" class="no-spinner" style="width: 70px"
                :value="rawFor(p.id)?.clickers[myClickerAssignment.slot]?.plus ?? 0"
                @input="saveClicker(p.id, 'plus', +($event.target as HTMLInputElement).value)"
              />
            </td>
            <td>
              <input
                type="number" min="0" class="no-spinner" style="width: 70px"
                :value="rawFor(p.id)?.clickers[myClickerAssignment.slot]?.minus ?? 0"
                @input="saveClicker(p.id, 'minus', +($event.target as HTMLInputElement).value)"
              />
            </td>
          </tr>
        </tbody>
      </table>
      </fieldset>
    </div>

    <div v-if="myEvalAssignment" class="card">
      <h2>Evaluation judge - slot {{ myEvalAssignment.slot }}</h2>
      <fieldset :disabled="inputsDisabled" style="border: 0; padding: 0; margin: 0">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Player</th>
            <th v-for="cat in categories" :key="cat.name">{{ cat.name }} (max {{ cat.maxValue }})</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in players" :key="p.id">
            <td>{{ p.number }}</td>
            <td>{{ p.name }}</td>
            <td v-for="cat in categories" :key="cat.name">
              <input
                type="number" min="0" :max="cat.maxValue" step="1" style="width: 70px"
                :value="rawFor(p.id)?.evals[myEvalAssignment.slot]?.[cat.name] ?? DEFAULT_EVAL_SCORE"
                @input="saveEval(p.id, cat.name, +($event.target as HTMLInputElement).value)"
              />
            </td>
          </tr>
        </tbody>
      </table>
      </fieldset>
    </div>

    <div v-if="canEditDeductions" class="card">
      <h2>Major deduction judge</h2>
      <p v-if="noMdAssigned" class="muted" style="margin-top: -6px">
        No major deduction judge assigned yet - entering as head judge by default.
      </p>
      <fieldset :disabled="inputsDisabled" style="border: 0; padding: 0; margin: 0">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Player</th>
            <th>Stop</th>
            <th>Discard</th>
            <th>{{ thirdDeductionLabel }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in players" :key="p.id">
            <td>{{ p.number }}</td>
            <td>{{ p.name }}</td>
            <td>
              <input
                type="number" min="0" style="width: 70px"
                :value="rawFor(p.id)?.deductions.stop ?? 0"
                @input="saveDeduction(p.id, 'stop', +($event.target as HTMLInputElement).value)"
              />
            </td>
            <td>
              <input
                type="number" min="0" style="width: 70px"
                :value="rawFor(p.id)?.deductions.discard ?? 0"
                @input="saveDeduction(p.id, 'discard', +($event.target as HTMLInputElement).value)"
              />
            </td>
            <td>
              <input
                type="number" min="0" style="width: 70px"
                :value="rawFor(p.id)?.deductions.cut ?? 0"
                @input="saveDeduction(p.id, 'cut', +($event.target as HTMLInputElement).value)"
              />
            </td>
          </tr>
        </tbody>
      </table>
      </fieldset>
    </div>

    <p v-if="players.length === 0" class="muted">No players in this division yet.</p>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
