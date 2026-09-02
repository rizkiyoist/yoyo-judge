<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { finalCategories } from '../lib/scoring'
import { useAuthStore } from '../stores/auth'
import { useContestStore } from '../stores/contests'
import type { Contest, Division, JudgeAssignment, PlayerResult, ScoringStage, User } from '../types'

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

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)
}

const STAGE_ORDER: ScoringStage[] = ['prelim', 'final']

function orderedStages(stages: ScoringStage[]): ScoringStage[] {
  return STAGE_ORDER.filter((s) => stages.includes(s))
}

function stageLabel(stage: ScoringStage): string {
  return stage === 'prelim' ? 'Prelim' : 'Final'
}

const auth = useAuthStore()
const store = useContestStore()
const newName = ref('')
const newYear = ref(new Date().getFullYear())
const creating = ref(false)

const userId = computed(() => auth.user?.id ?? '')
const judgesByContest = ref<Record<string, JudgeAssignment[]>>({})
const usersById = ref<Record<string, User>>({})
const resultsByDivisionStage = ref<Record<string, PlayerResult[]>>({})

onMounted(async () => {
  if (userId.value) await store.fetchContests()
})

function resultsKey(divisionId: string, stage: ScoringStage): string {
  return `${divisionId}:${stage}`
}

function topThree(divisionId: string, stage: ScoringStage): PlayerResult[] {
  return (resultsByDivisionStage.value[resultsKey(divisionId, stage)] ?? [])
    .slice()
    .sort((a, b) => a.place - b.place)
    .slice(0, 3)
}

// Final is the definitive result; only fall back to prelim's top 3 when a
// division has no final stage at all.
function topThreeStage(stages: ScoringStage[]): ScoringStage | null {
  if (stages.includes('final')) return 'final'
  if (stages.includes('prelim')) return 'prelim'
  return null
}

// Load each contest's judge assignments (to show "Judges: ...") and each
// division+stage's results (to show the top 3) once the contest list is
// known, and re-run whenever it changes (e.g. after create).
watch(
  () => store.contests,
  async (contests: Contest[]) => {
    const ids = new Set(contests.map((c) => c.ownerUserId))
    for (const contest of contests) {
      judgesByContest.value[contest.id] = await api.listJudgeAssignments(contest.id)
      for (const a of judgesByContest.value[contest.id]) ids.add(a.userId)
      for (const division of contest.divisions) {
        for (const stage of division.stages) {
          resultsByDivisionStage.value[resultsKey(division.id, stage)] = await api.getResults(division.id, stage)
        }
      }
    }
    const users = await api.getUsers([...ids])
    usersById.value = Object.fromEntries(users.map((u) => [u.id, u]))
  },
  { deep: false },
)

function judgeName(userId: string): string {
  const u = usersById.value[userId]
  if (!u) return userId
  return `${u.firstName} ${u.lastName}`.trim() || `${u.email} (not signed in yet)`
}

// A judge assigned to both prelim and final gets one JudgeAssignment per
// stage; collapse those into one entry per (user, role, slot) for display,
// sorted by slot.
function judgesByRole(assignments: JudgeAssignment[] | undefined, role: JudgeAssignment['role']): JudgeAssignment[] {
  const seen = new Set<string>()
  return (assignments ?? [])
    .filter((a) => a.role === role)
    .filter((a) => {
      const key = `${a.userId}:${a.slot}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
    .sort((a, b) => a.slot - b.slot)
}

const downloading = ref<Record<string, boolean>>({})
const lockToggling = ref<Record<string, boolean>>({})

async function toggleLock(contest: Contest) {
  lockToggling.value[contest.id] = true
  try {
    await api.setContestLocked(contest.id, !contest.locked)
    await store.fetchContests()
  } finally {
    lockToggling.value[contest.id] = false
  }
}

function divisionResultsTable(division: Division, results: PlayerResult[]): string {
  const categories = finalCategories()
  const categoryHeaders = categories.map((c) => `<th>${escapeHtml(FINAL_CATEGORY_LABELS[c.name] ?? c.name)}</th>`).join('')
  const rows = [...results]
    .sort((a, b) => a.place - b.place)
    .map((r) => {
      const categoryCells = categories
        .map((c) => `<td>${(r.categoryScores[c.name] ?? 0).toFixed(2)}</td>`)
        .join('')
      const categoriesTotal = (r.groupTotals.TEv ?? 0) + (r.groupTotals.PEv ?? 0)
      const thirdDeduction = r.deductionTotals.Cut ?? r.deductionTotals.Detach ?? 0
      return `<tr>
        <td>${r.place}</td>
        <td>${escapeHtml(r.name)}</td>
        <td>${r.technicalExecution.toFixed(2)}</td>
        ${categoryCells}
        <td>${categoriesTotal.toFixed(2)}</td>
        <td>${r.evaluationTotal.toFixed(2)}</td>
        <td>-${(r.deductionTotals.Stop ?? 0).toFixed(2)}</td>
        <td>-${(r.deductionTotals.Discard ?? 0).toFixed(2)}</td>
        <td>-${thirdDeduction.toFixed(2)}</td>
        <td><strong>${r.finalScore.toFixed(2)}</strong></td>
      </tr>`
    })
    .join('\n')

  return `<h2>${escapeHtml(division.name)}</h2>
  <table>
    <thead>
      <tr>
        <th>Place</th><th>Player</th><th>TE</th>${categoryHeaders}
        <th>Categories Total</th><th>E.Total</th><th>Stop</th><th>Discard</th><th>Cut</th><th>Final Score</th>
      </tr>
    </thead>
    <tbody>
      ${rows || '<tr><td colspan="11">No results yet.</td></tr>'}
    </tbody>
  </table>`
}

function triggerHtmlDownload(filename: string, html: string) {
  const blob = new Blob([html], { type: 'text/html' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

async function downloadContestResults(contest: Contest) {
  downloading.value[contest.id] = true
  try {
    const finalDivisions = contest.divisions.filter((d) => d.stages.includes('final'))
    const sections = await Promise.all(
      finalDivisions.map(async (division) => {
        const results = await api.getResults(division.id, 'final')
        return divisionResultsTable(division, results)
      }),
    )
    const html = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>${escapeHtml(contest.name)} - Final Results</title>
<style>
  body { font: 14px/1.5 system-ui, sans-serif; padding: 24px; color: #2b2b33; }
  h1 { margin-bottom: 4px; }
  h2 { margin-top: 32px; }
  table { border-collapse: collapse; width: 100%; margin-top: 8px; }
  th, td { border: 1px solid #e0dee4; padding: 6px 10px; text-align: left; font-size: 13px; }
  th { background: #f6f5f9; }
</style>
</head>
<body>
<h1>${escapeHtml(contest.name)} (${contest.year})</h1>
<p>Final results - all divisions</p>
${sections.join('\n') || '<p>No divisions with a final stage.</p>'}
</body>
</html>`
    const filename = `${contest.name.replace(/[^a-z0-9]+/gi, '-').toLowerCase()}-final-results.html`
    triggerHtmlDownload(filename, html)
  } finally {
    downloading.value[contest.id] = false
  }
}

async function createContest() {
  if (!newName.value.trim() || !newYear.value) return
  creating.value = true
  try {
    await store.createContest(newName.value.trim(), newYear.value, userId.value)
    newName.value = ''
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <h1>All Contests</h1>

  <div class="card">
    <h2>Create a contest</h2>
    <form class="row" @submit.prevent="createContest">
      <input v-model="newName" type="text" placeholder="Contest name" style="flex: 1" />
      <input v-model.number="newYear" type="number" placeholder="Year" style="width: 100px" />
      <button class="primary" type="submit" :disabled="creating">Create</button>
    </form>
  </div>

  <p v-if="store.loading" class="muted">Loading…</p>

  <div v-for="contest in store.contests" :key="contest.id" class="card">
    <div class="row" style="justify-content: space-between; margin-bottom: 16px">
      <div class="row">
        <h2 style="margin: 0">{{ contest.name }}</h2>
        <span class="badge-year">{{ contest.year }}</span>
        <span v-if="contest.locked" class="badge" title="Locked by head judge">🔒 Locked</span>
      </div>
      <div class="row">
        <button
          v-if="contest.headJudgeUserId === userId"
          :disabled="lockToggling[contest.id]"
          :class="{ primary: !contest.locked }"
          :title="contest.locked ? 'Unlock this contest so scores and settings can be changed again.' : 'Lock this contest to freeze all scores and settings.'"
          @click="toggleLock(contest)"
        >
          {{ lockToggling[contest.id] ? 'Working…' : contest.locked ? 'Unlock' : 'Lock' }}
        </button>
        <RouterLink :to="{ name: 'contest-edit', params: { contestId: contest.id } }">
          <button>Divisions</button>
        </RouterLink>
        <RouterLink :to="{ name: 'contest-judges', params: { contestId: contest.id } }">
          <button>Judges</button>
        </RouterLink>
        <button :disabled="downloading[contest.id]" @click="downloadContestResults(contest)">
          {{ downloading[contest.id] ? 'Preparing…' : 'Download Results' }}
        </button>
      </div>
    </div>

    <p v-if="!contest.divisions.length" class="muted">No divisions yet - add one to get started.</p>

    <table v-else>
      <thead>
        <tr>
          <th>Division</th>
          <th>Stages</th>
          <th>Players</th>
          <th>Input Score</th>
          <th>Result Detail</th>
          <th>Top 3</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="division in contest.divisions" :key="division.id">
          <td>{{ division.name }}</td>
          <td>
            <span v-for="stage in division.stages" :key="stage" class="badge" style="margin-right: 4px">
              {{ stage }}
            </span>
          </td>
          <td>
            <RouterLink :to="{ name: 'division-players', params: { contestId: contest.id, divisionId: division.id } }">
              <button>Players</button>
            </RouterLink>
          </td>
          <td>
            <div style="display: flex; flex-direction: column; gap: 6px; align-items: stretch">
              <RouterLink
                v-for="stage in orderedStages(division.stages)"
                :key="stage"
                :to="{ name: 'score-entry', params: { contestId: contest.id, divisionId: division.id, stage } }"
              >
                <button style="width: 100%">{{ stageLabel(stage) }}</button>
              </RouterLink>
            </div>
          </td>
          <td>
            <div style="display: flex; flex-direction: column; gap: 6px; align-items: stretch">
              <RouterLink
                v-for="stage in orderedStages(division.stages)"
                :key="stage + '-results'"
                :to="{ name: 'results', params: { contestId: contest.id, divisionId: division.id, stage } }"
              >
                <button style="width: 100%">{{ stageLabel(stage) }}</button>
              </RouterLink>
            </div>
          </td>
          <td>
            <template v-if="topThreeStage(division.stages)">
              <div class="muted" style="font-size: 0.78rem">{{ stageLabel(topThreeStage(division.stages)!) }}</div>
              <ol
                v-if="topThree(division.id, topThreeStage(division.stages)!).length"
                style="margin: 0; padding-left: 16px"
              >
                <li v-for="r in topThree(division.id, topThreeStage(division.stages)!)" :key="r.playerId">
                  {{ r.name }}
                </li>
              </ol>
              <span v-else class="muted">-</span>
            </template>
            <span v-else class="muted">-</span>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="judgesByContest[contest.id]?.length" class="row" style="align-items: flex-start; margin-top: 14px; gap: 32px">
      <div>
        <h3 style="margin: 0 0 4px">Clicker Judges (TEx)</h3>
        <p v-if="!judgesByRole(judgesByContest[contest.id], 'clicker').length" class="muted">None assigned.</p>
        <ul v-else style="margin: 0; padding-left: 18px">
          <li v-for="a in judgesByRole(judgesByContest[contest.id], 'clicker')" :key="a.id">
            #{{ a.slot }} -
            <strong v-if="a.userId === contest.ownerUserId">{{ judgeName(a.userId) }}</strong>
            <template v-else>{{ judgeName(a.userId) }}</template>
          </li>
        </ul>
      </div>
      <div>
        <h3 style="margin: 0 0 4px">Evaluation Judges (PEv)</h3>
        <p v-if="!judgesByRole(judgesByContest[contest.id], 'evaluator').length" class="muted">None assigned.</p>
        <ul v-else style="margin: 0; padding-left: 18px">
          <li v-for="a in judgesByRole(judgesByContest[contest.id], 'evaluator')" :key="a.id">
            #{{ a.slot }} -
            <strong v-if="a.userId === contest.ownerUserId">{{ judgeName(a.userId) }}</strong>
            <template v-else>{{ judgeName(a.userId) }}</template>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <p v-if="!store.loading && !store.contests.length" class="muted">
    No contests yet. Create one above, or ask a head judge to invite you.
  </p>
</template>
