<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
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
const saving = ref<Record<string, boolean>>({})

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

async function saveClicker(playerId: string, field: 'plus' | 'minus', value: number) {
  if (!activeClickerAssignment.value) return
  const slot = activeClickerAssignment.value.slot
  const existing = rawFor(playerId)?.clickers[slot] ?? { plus: 0, minus: 0 }
  const next = { ...existing, [field]: value }
  saving.value[playerId] = true
  try {
    await api.submitClickerScore(props.divisionId, props.stage, playerId, slot, next)
    await load()
  } finally {
    saving.value[playerId] = false
  }
}

async function saveDeduction(playerId: string, field: 'stop' | 'discard' | 'cut', value: number) {
  const existing = rawFor(playerId)?.deductions ?? { stop: 0, discard: 0, cut: 0 }
  const next = { ...existing, [field]: value }
  saving.value[playerId] = true
  try {
    await api.submitDeductions(props.divisionId, props.stage, playerId, next)
    await load()
  } finally {
    saving.value[playerId] = false
  }
}

async function saveEval(playerId: string, category: string, value: number) {
  if (!activeEvalAssignment.value) return
  const slot = activeEvalAssignment.value.slot
  const existing = rawFor(playerId)?.evals[slot] ?? {}
  const next = { ...existing, [category]: value }
  saving.value[playerId] = true
  try {
    await api.submitEvalScore(props.divisionId, props.stage, playerId, slot, next)
    await load()
  } finally {
    saving.value[playerId] = false
  }
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
        Enter or correct any clicker or evaluation judge's raw scores on their behalf. Pick the judge to act as below.
      </p>

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
                  :value="rawFor(p.id)?.clickers[activeClickerAssignment.slot]?.plus ?? 0"
                  @change="saveClicker(p.id, 'plus', +($event.target as HTMLInputElement).value)"
                />
              </td>
              <td>
                <input
                  type="number" min="0" style="width: 70px"
                  :value="rawFor(p.id)?.clickers[activeClickerAssignment.slot]?.minus ?? 0"
                  @change="saveClicker(p.id, 'minus', +($event.target as HTMLInputElement).value)"
                />
              </td>
              <td>
                <input
                  type="number" min="0" style="width: 70px"
                  :value="rawFor(p.id)?.deductions.stop ?? 0"
                  @change="saveDeduction(p.id, 'stop', +($event.target as HTMLInputElement).value)"
                />
              </td>
              <td>
                <input
                  type="number" min="0" style="width: 70px"
                  :value="rawFor(p.id)?.deductions.discard ?? 0"
                  @change="saveDeduction(p.id, 'discard', +($event.target as HTMLInputElement).value)"
                />
              </td>
              <td>
                <input
                  type="number" min="0" style="width: 70px"
                  :value="rawFor(p.id)?.deductions.cut ?? 0"
                  @change="saveDeduction(p.id, 'cut', +($event.target as HTMLInputElement).value)"
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
                <input
                  type="number" min="0" :max="cat.maxValue" step="1" style="width: 70px"
                  :value="rawFor(p.id)?.evals[activeEvalAssignment.slot]?.[cat.name] ?? 0"
                  @change="saveEval(p.id, cat.name, +($event.target as HTMLInputElement).value)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="muted">No evaluation judges assigned to this division/stage yet.</p>
    </template>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
