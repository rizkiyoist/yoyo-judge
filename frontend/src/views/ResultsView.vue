<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { finalCategories, prelimCategories } from '../lib/scoring'
import type { Contest, JudgeAssignment, Player, PlayerRawScores, PlayerResult, ScoringStage, User } from '../types'

const props = defineProps<{ contestId: string; divisionId: string; stage: ScoringStage }>()

const contest = ref<Contest | null>(null)
const players = ref<Player[]>([])
const assignments = ref<JudgeAssignment[]>([])
const rawScores = ref<PlayerRawScores[]>([])
const results = ref<PlayerResult[]>([])
const usersById = ref<Record<string, User>>({})
const expanded = ref<Record<string, boolean>>({})

const division = computed(() => contest.value?.divisions.find((d) => d.id === props.divisionId))
const categories = computed(() => (props.stage === 'prelim' ? prelimCategories() : finalCategories()))
const thirdDeductionLabel = computed(() => (props.stage === 'prelim' ? 'Detach' : 'Cut'))

// Descriptive column labels mirroring the original xls workbook's RESULT /
// FINAL-SCORE sheets, rather than the raw internal category codes.
const FINAL_CATEGORY_LABELS: Record<string, string> = {
  EXE: 'Execution',
  CTL: 'Control',
  TDV: 'Trick Diversity',
  SEM: 'Space Use/Emp.',
  MU1: 'Choreography',
  MU2: 'Construction',
  BDY: 'Body Control',
  SHW: 'Showmanship',
}
const PRELIM_CATEGORY_LABELS: Record<string, string> = {
  EXE: 'Cleanliness',
  CTL: 'Execution',
  MU1: 'Music Use',
  BDY: 'Body Control',
}
const categoryLabels = computed(() => (props.stage === 'prelim' ? PRELIM_CATEGORY_LABELS : FINAL_CATEGORY_LABELS))

function categoriesTotal(r: PlayerResult): number {
  return (r.groupTotals.TEv ?? 0) + (r.groupTotals.PEv ?? 0)
}

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

const sortedResults = computed(() => [...results.value].sort((a, b) => a.place - b.place))

async function load() {
  contest.value = await api.getContest(props.contestId)
  players.value = await api.listPlayers(props.divisionId)
  assignments.value = await api.listJudgeAssignments(props.contestId)
  rawScores.value = await api.getRawScores(props.divisionId, props.stage)
  results.value = await api.getResults(props.divisionId, props.stage)
  const ids = new Set(assignments.value.map((a) => a.userId))
  if (contest.value) ids.add(contest.value.ownerUserId)
  const users = await api.getUsers([...ids])
  usersById.value = Object.fromEntries(users.map((u) => [u.id, u]))
}

onMounted(load)

function judgeLabel(userId: string | undefined): string {
  if (!userId) return '-'
  const u = usersById.value[userId]
  if (!u) return userId
  return `${u.firstName} ${u.lastName}`.trim() || `${u.email} (not signed in yet)`
}

function rawFor(playerId: string): PlayerRawScores | undefined {
  return rawScores.value.find((r) => r.playerId === playerId)
}

function toggle(playerId: string) {
  expanded.value[playerId] = !expanded.value[playerId]
}
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <h1>{{ contest.name }} - {{ division?.name }} ({{ stage }}) results</h1>

    <div class="card">
      <table v-if="sortedResults.length" style="min-width: max-content">
        <thead>
          <tr>
            <th>Place</th>
            <th>#</th>
            <th>Player</th>
            <th class="col-tex">TE</th>
            <th v-for="cat in categories" :key="cat.name" :class="cat.group === 'TEv' ? 'col-tex' : 'col-pev'">
              {{ categoryLabels[cat.name] ?? cat.name }}
            </th>
            <th>Categories Total</th>
            <th>E.Total</th>
            <th>Stop</th>
            <th>Discard</th>
            <th>{{ thirdDeductionLabel }}</th>
            <th>Final Score</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <template v-for="r in sortedResults" :key="r.playerId">
            <tr>
              <td :class="'place-' + r.place">{{ r.place }}</td>
              <td>{{ players.find((p) => p.id === r.playerId)?.number }}</td>
              <td>{{ r.name }}</td>
              <td class="col-tex">{{ r.technicalExecution.toFixed(2) }}</td>
              <td v-for="cat in categories" :key="cat.name" :class="cat.group === 'TEv' ? 'col-tex' : 'col-pev'">
                {{ (r.categoryScores[cat.name] ?? 0).toFixed(2) }}
              </td>
              <td>{{ categoriesTotal(r).toFixed(2) }}</td>
              <td>{{ r.evaluationTotal.toFixed(2) }}</td>
              <td>-{{ (r.deductionTotals.Stop ?? 0).toFixed(2) }}</td>
              <td>-{{ (r.deductionTotals.Discard ?? 0).toFixed(2) }}</td>
              <td>-{{ (r.deductionTotals.Cut ?? r.deductionTotals.Detach ?? 0).toFixed(2) }}</td>
              <td><strong>{{ r.finalScore.toFixed(2) }}</strong></td>
              <td><button @click="toggle(r.playerId)">{{ expanded[r.playerId] ? 'Hide' : 'Details' }}</button></td>
            </tr>
            <tr v-if="expanded[r.playerId]">
              <td :colspan="11 + categories.length">
                <h3>Clicker judges (M.D. - Stop {{ rawFor(r.playerId)?.deductions.stop ?? 0 }}, Discard
                  {{ rawFor(r.playerId)?.deductions.discard ?? 0 }}, {{ thirdDeductionLabel }}
                  {{ rawFor(r.playerId)?.deductions.cut ?? 0 }})</h3>
                <table>
                  <thead>
                    <tr>
                      <th>Slot</th>
                      <th>Judge</th>
                      <th>+</th>
                      <th>-</th>
                      <th>Net</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="a in clickerAssignments" :key="a.id">
                      <td>{{ a.slot }}</td>
                      <td>{{ judgeLabel(a.userId) }}</td>
                      <td>{{ rawFor(r.playerId)?.clickers[a.slot]?.plus ?? 0 }}</td>
                      <td>{{ rawFor(r.playerId)?.clickers[a.slot]?.minus ?? 0 }}</td>
                      <td>
                        {{ (rawFor(r.playerId)?.clickers[a.slot]?.plus ?? 0) - (rawFor(r.playerId)?.clickers[a.slot]?.minus ?? 0) }}
                      </td>
                    </tr>
                  </tbody>
                </table>

                <h3>Evaluation judges</h3>
                <table>
                  <thead>
                    <tr>
                      <th>Slot</th>
                      <th>Judge</th>
                      <th v-for="cat in categories" :key="cat.name">{{ cat.name }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="a in evalAssignments" :key="a.id">
                      <td>{{ a.slot }}</td>
                      <td>{{ judgeLabel(a.userId) }}</td>
                      <td v-for="cat in categories" :key="cat.name">
                        {{ rawFor(r.playerId)?.evals[a.slot]?.[cat.name] ?? 0 }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
      <p v-else class="muted">No players to score yet.</p>
    </div>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
