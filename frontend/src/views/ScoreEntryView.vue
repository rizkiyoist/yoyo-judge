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
const saving = ref<Record<string, boolean>>({})

const division = computed(() => contest.value?.divisions.find((d) => d.id === props.divisionId))
const isOwner = computed(() => contest.value?.ownerUserId === auth.user?.id)
const stageLocked = computed(() => division.value?.lockedStages.includes(props.stage) ?? false)
// The owner is exempt from the lock (that's the whole point of locking —
// stopping *other* judges; the head judge can still fix things via override).
const inputsDisabled = computed(() => stageLocked.value && !isOwner.value)

const myAssignments = computed(() =>
  assignments.value.filter(
    (a) => a.divisionId === props.divisionId && a.stage === props.stage && a.userId === auth.user?.id,
  ),
)
const myClickerAssignment = computed(() => myAssignments.value.find((a) => a.role === 'clicker'))
const myEvalAssignment = computed(() => myAssignments.value.find((a) => a.role === 'evaluator'))

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

const lockToggling = ref(false)
const lockError = ref('')

async function toggleLock() {
  lockToggling.value = true
  lockError.value = ''
  try {
    await api.setDivisionStageLock(props.divisionId, props.stage, !stageLocked.value)
    await load()
  } catch (e) {
    lockError.value = e instanceof Error ? e.message : 'Failed to update lock'
  } finally {
    lockToggling.value = false
  }
}

async function saveClicker(playerId: string, field: 'plus' | 'minus', value: number) {
  if (!myClickerAssignment.value || inputsDisabled.value) return
  const slot = myClickerAssignment.value.slot
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
  if (inputsDisabled.value) return
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
  if (!myEvalAssignment.value || inputsDisabled.value) return
  const slot = myEvalAssignment.value.slot
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
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <h1>{{ contest.name }} — {{ division?.name }} ({{ stage }}) scoring</h1>

    <p v-if="isOwner" class="row" style="align-items: center">
      <RouterLink :to="{ name: 'score-override', params: { contestId, divisionId, stage } }">
        <button>Head judge: override any judge's score</button>
      </RouterLink>
      <button :class="{ primary: !stageLocked }" :disabled="lockToggling" @click="toggleLock">
        {{ lockToggling ? 'Working…' : stageLocked ? 'Unlock scores' : 'Lock scores' }}
      </button>
      <span v-if="lockError" class="error">{{ lockError }}</span>
    </p>

    <p v-if="stageLocked" class="muted">
      🔒 Scores for this stage are locked by the head judge{{ isOwner ? '' : ' and can no longer be changed' }}.
    </p>

    <p v-if="!myAssignments.length" class="muted">
      You aren't assigned as a judge for this division/stage. Ask the head judge to invite you from the Judges page.
    </p>

    <div v-if="myClickerAssignment" class="card">
      <h2>Clicker judge — slot {{ myClickerAssignment.slot }}</h2>
      <fieldset :disabled="inputsDisabled" style="border: 0; padding: 0; margin: 0">
      <table>
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
                :value="rawFor(p.id)?.clickers[myClickerAssignment.slot]?.plus ?? 0"
                @change="saveClicker(p.id, 'plus', +($event.target as HTMLInputElement).value)"
              />
            </td>
            <td>
              <input
                type="number" min="0" style="width: 70px"
                :value="rawFor(p.id)?.clickers[myClickerAssignment.slot]?.minus ?? 0"
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
      </fieldset>
      <p class="muted">
        Major deductions (Stop/Discard/{{ thirdDeductionLabel }}) apply once per player — coordinate with the other
        clicker judges so they aren't entered twice.
      </p>
    </div>

    <div v-if="myEvalAssignment" class="card">
      <h2>Evaluation judge — slot {{ myEvalAssignment.slot }}</h2>
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
                type="number" min="0" :max="cat.maxValue" step="0.5" style="width: 70px"
                :value="rawFor(p.id)?.evals[myEvalAssignment.slot]?.[cat.name] ?? 0"
                @change="saveEval(p.id, cat.name, +($event.target as HTMLInputElement).value)"
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
