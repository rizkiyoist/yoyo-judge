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
  const allUsers = await api.searchUsers('example.com')
  usersById.value = Object.fromEntries(allUsers.map((u) => [u.id, u]))
}

onMounted(load)

function judgeLabel(userId: string | undefined): string {
  if (!userId) return '—'
  const u = usersById.value[userId]
  return u ? `${u.firstName} ${u.lastName}` : userId
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
    <h1>{{ contest.name }} — {{ division?.name }} ({{ stage }}) results</h1>

    <div class="card">
      <table v-if="sortedResults.length">
        <thead>
          <tr>
            <th>Place</th>
            <th>#</th>
            <th>Player</th>
            <th>T.Ex</th>
            <th>T.Ev</th>
            <th>P.Ev</th>
            <th>Eval Total</th>
            <th>Deductions</th>
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
              <td>{{ r.technicalExecution.toFixed(2) }}</td>
              <td>{{ (r.groupTotals.TEv ?? 0).toFixed(2) }}</td>
              <td>{{ (r.groupTotals.PEv ?? 0).toFixed(2) }}</td>
              <td>{{ r.evaluationTotal.toFixed(2) }}</td>
              <td>-{{ r.deductionTotal.toFixed(2) }}</td>
              <td><strong>{{ r.finalScore.toFixed(2) }}</strong></td>
              <td><button @click="toggle(r.playerId)">{{ expanded[r.playerId] ? 'Hide' : 'Details' }}</button></td>
            </tr>
            <tr v-if="expanded[r.playerId]">
              <td colspan="10">
                <h3>Clicker judges (M.D. — Stop {{ rawFor(r.playerId)?.deductions.stop ?? 0 }}, Discard
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
