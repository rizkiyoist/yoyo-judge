<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import ScoreNumberInput from '../components/ScoreNumberInput.vue'
import { finalCategories, prelimCategories } from '../lib/scoring'
import { useAuthStore } from '../stores/auth'
import type { Contest, JudgeAssignment, Player, PlayerRawScores, ScoringStage, User } from '../types'

const props = defineProps<{ contestId: string; divisionId: string; stage: ScoringStage }>()

const auth = useAuthStore()
const contest = ref<Contest | null>(null)
const players = ref<Player[]>([])
const assignments = ref<JudgeAssignment[]>([])
const rawScores = ref<PlayerRawScores[]>([])
const usersById = ref<Record<string, User>>({})
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

const clickerAssignments = computed(() =>
  assignments.value
    .filter((a) => a.divisionId === props.divisionId && a.stage === props.stage && a.role === 'clicker')
    .sort((a, b) => a.slot - b.slot),
)
const evalAssignments = computed(() =>
  assignments.value
    .filter((a) => a.divisionId === props.divisionId && a.stage === props.stage && a.role === 'evaluator')
    .sort((a, b) => a.slot - b.slot),
)
const mdAssignment = computed(() =>
  assignments.value.find(
    (a) => a.divisionId === props.divisionId && a.stage === props.stage && a.role === 'major_deduction',
  ),
)

const overrideClickerUserId = ref('')
const overrideEvalUserId = ref('')
watch(clickerAssignments, (list) => {
  if (!list.some((a) => a.userId === overrideClickerUserId.value)) {
    overrideClickerUserId.value = list[0]?.userId ?? ''
  }
})
watch(evalAssignments, (list) => {
  if (!list.some((a) => a.userId === overrideEvalUserId.value)) {
    overrideEvalUserId.value = list[0]?.userId ?? ''
  }
})

const activeClickerAssignment = computed(() => clickerAssignments.value.find((a) => a.userId === overrideClickerUserId.value))
const activeEvalAssignment = computed(() => evalAssignments.value.find((a) => a.userId === overrideEvalUserId.value))

const categories = computed(() => (props.stage === 'prelim' ? prelimCategories() : finalCategories()))
const thirdDeductionLabel = computed(() => (props.stage === 'prelim' ? 'Detach' : 'Cut'))

async function load() {
  contest.value = await api.getContest(props.contestId)
  players.value = await api.listPlayers(props.divisionId)
  assignments.value = await api.listJudgeAssignments(props.contestId)
  rawScores.value = await api.getRawScores(props.divisionId, props.stage)
  const ids = new Set(assignments.value.map((a) => a.userId))
  if (contest.value) ids.add(contest.value.ownerUserId)
  const users = await api.getUsers([...ids])
  usersById.value = Object.fromEntries(users.map((u) => [u.id, u]))
}

onMounted(load)

function rawFor(playerId: string): PlayerRawScores | undefined {
  return rawScores.value.find((r) => r.playerId === playerId)
}

function judgeName(userId: string): string {
  const u = usersById.value[userId]
  if (!u) return userId
  return `${u.firstName} ${u.lastName}`.trim() || `${u.email} (not signed in yet)`
}

function saveClicker(playerId: string, field: 'plus' | 'minus', value: number) {
  if (!activeClickerAssignment.value) return
  const key = statusKey('clicker', playerId, field)
  debounceSave(key, async () => {
    const slot = activeClickerAssignment.value!.slot
    const existing = rawFor(playerId)?.clickers[slot] ?? { plus: 0, minus: 0 }
    const next = { ...existing, [field]: value }
    await api.submitClickerScore(props.divisionId, props.stage, playerId, slot, next)
    await load()
  })
}

function saveDeduction(playerId: string, field: 'stop' | 'discard' | 'cut', value: number) {
  const key = statusKey('deduction', playerId, field)
  debounceSave(key, async () => {
    const existing = rawFor(playerId)?.deductions ?? { stop: 0, discard: 0, cut: 0 }
    const next = { ...existing, [field]: value }
    await api.submitDeductions(props.divisionId, props.stage, playerId, next)
    await load()
  })
}

function saveEval(playerId: string, category: string, value: number) {
  if (!activeEvalAssignment.value) return
  const key = statusKey('eval', playerId, category)
  debounceSave(key, async () => {
    const slot = activeEvalAssignment.value!.slot
    const existing = rawFor(playerId)?.evals[slot] ?? {}
    const next = { ...existing, [category]: value }
    await api.submitEvalScore(props.divisionId, props.stage, playerId, slot, next)
    await load()
  })
}
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'score-entry', params: { contestId, divisionId, stage } }">&larr; Back to scoring</RouterLink>
    <h1>{{ contest.name }} - {{ division?.name }} ({{ stage }}) - head judge override</h1>

    <p v-if="!isHeadJudge" class="error">Only the head judge can override judge scores.</p>
    <p v-else-if="contest.locked" class="error">
      🔒 This contest is locked - unlock it from the Judges page before overriding any scores.
    </p>

    <template v-else>
      <p class="muted">
        Enter or correct any judge's raw scores on their behalf, including major deductions.
      </p>

      <div v-if="saveState !== 'idle'" class="save-status-bar">
        <span
          class="save-status"
          :class="saveState === 'saved' ? 'save-status--saved' : 'save-status--pending'"
        >
          {{ saveState === 'saved' ? 'All changes saved' : 'Saving…' }}
        </span>
      </div>

      <div v-if="clickerAssignments.length" class="card">
        <div class="row" style="justify-content: space-between">
          <h2 style="margin: 0">Clicker judge scores</h2>
          <div class="field" style="margin: 0">
            <label>Acting as</label>
            <select v-model="overrideClickerUserId">
              <option v-for="a in clickerAssignments" :key="a.id" :value="a.userId">
                #{{ a.slot }} - {{ judgeName(a.userId) }}
              </option>
            </select>
          </div>
        </div>
        <table v-if="activeClickerAssignment">
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
                <ScoreNumberInput
                  :min="0"
                  :model-value="rawFor(p.id)?.clickers[activeClickerAssignment.slot]?.plus ?? 0"
                  @update:model-value="saveClicker(p.id, 'plus', $event)"
                />
              </td>
              <td>
                <ScoreNumberInput
                  :min="0"
                  :model-value="rawFor(p.id)?.clickers[activeClickerAssignment.slot]?.minus ?? 0"
                  @update:model-value="saveClicker(p.id, 'minus', $event)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="muted">No clicker judges assigned to this division/stage yet.</p>

      <div v-if="evalAssignments.length" class="card">
        <div class="row" style="justify-content: space-between">
          <h2 style="margin: 0">Evaluation judge scores</h2>
          <div class="field" style="margin: 0">
            <label>Acting as</label>
            <select v-model="overrideEvalUserId">
              <option v-for="a in evalAssignments" :key="a.id" :value="a.userId">
                #{{ a.slot }} - {{ judgeName(a.userId) }}
              </option>
            </select>
          </div>
        </div>
        <table v-if="activeEvalAssignment">
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
                <ScoreNumberInput
                  :min="0"
                  :max="cat.maxValue"
                  :model-value="rawFor(p.id)?.evals[activeEvalAssignment.slot]?.[cat.name] ?? DEFAULT_EVAL_SCORE"
                  @update:model-value="saveEval(p.id, cat.name, $event)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="muted">No evaluation judges assigned to this division/stage yet.</p>

      <div v-if="mdAssignment" class="card">
        <h2 style="margin: 0 0 10px">Major deduction judge scores</h2>
        <p class="muted" style="margin-top: -6px">Acting as {{ judgeName(mdAssignment.userId) }}.</p>
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
                <ScoreNumberInput
                  :min="0"
                  :model-value="rawFor(p.id)?.deductions.stop ?? 0"
                  @update:model-value="saveDeduction(p.id, 'stop', $event)"
                />
              </td>
              <td>
                <ScoreNumberInput
                  :min="0"
                  :model-value="rawFor(p.id)?.deductions.discard ?? 0"
                  @update:model-value="saveDeduction(p.id, 'discard', $event)"
                />
              </td>
              <td>
                <ScoreNumberInput
                  :min="0"
                  :model-value="rawFor(p.id)?.deductions.cut ?? 0"
                  @update:model-value="saveDeduction(p.id, 'cut', $event)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="muted">No major deduction judge assigned to this division/stage yet.</p>
    </template>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
