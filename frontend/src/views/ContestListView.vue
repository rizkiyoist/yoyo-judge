<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { finalCategories } from '../lib/scoring'
import { useAuthStore } from '../stores/auth'
import { useContestStore } from '../stores/contests'
import Icon from '../components/Icon.vue'
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

// Accordion state — a contest expands to reveal its per-division-per-stage
// top-3 preview. Default: first contest opens on load.
const openContests = ref<Set<string>>(new Set())
function isOpen(contestId: string): boolean {
  return openContests.value.has(contestId)
}
function toggleOpen(contestId: string): void {
  const next = new Set(openContests.value)
  if (next.has(contestId)) next.delete(contestId)
  else next.add(contestId)
  openContests.value = next
}
// After the initial fetch, auto-expand the first contest so the page isn't
// a wall of collapsed rows on first visit.
watch(
  () => store.contests,
  (contests) => {
    if (!openContests.value.size && contests.length) {
      openContests.value = new Set([contests[0].id])
    }
  },
  { immediate: true },
)

const downloading = ref<Record<string, boolean>>({})
const lockToggling = ref<Record<string, boolean>>({})
const hidingToggling = ref<Record<string, boolean>>({})
const editingContestId = ref<string | null>(null)
const editName = ref('')
const editYear = ref(new Date().getFullYear())
const savingEdit = ref(false)

async function toggleLock(contest: Contest) {
  lockToggling.value[contest.id] = true
  try {
    await api.setContestLocked(contest.id, !contest.locked)
    await store.fetchContests()
  } finally {
    lockToggling.value[contest.id] = false
  }
}

async function toggleHidden(contest: Contest) {
  hidingToggling.value[contest.id] = true
  try {
    await store.setContestHidden(contest.id, !contest.hidden)
  } finally {
    hidingToggling.value[contest.id] = false
  }
}

function startEditContest(contest: Contest) {
  editingContestId.value = contest.id
  editName.value = contest.name
  editYear.value = contest.year
}

function cancelEditContest() {
  editingContestId.value = null
}

async function saveContestEdit(contest: Contest) {
  if (!editName.value.trim() || !editYear.value) return
  savingEdit.value = true
  try {
    await store.updateContest(contest.id, editName.value.trim(), editYear.value)
    editingContestId.value = null
  } finally {
    savingEdit.value = false
  }
}

function divisionResultsTable(division: Division, results: PlayerResult[]): string {
  const categories = finalCategories()
  const tevCategories = categories.filter((c) => c.group === 'TEv')
  const pevCategories = categories.filter((c) => c.group === 'PEv')
  const tevHeaders = tevCategories
    .map((c) => `<th class="col-tev">${escapeHtml(FINAL_CATEGORY_LABELS[c.name] ?? c.name)}</th>`)
    .join('')
  const pevHeaders = pevCategories
    .map((c) => `<th class="col-pev">${escapeHtml(FINAL_CATEGORY_LABELS[c.name] ?? c.name)}</th>`)
    .join('')
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
        <th rowspan="2">Place</th><th rowspan="2">Player</th><th class="col-tex" rowspan="2">T.Ex</th>
        <th class="col-tev" colspan="${tevCategories.length}">T.Ev</th>
        <th class="col-pev" colspan="${pevCategories.length}">P.Ev</th>
        <th rowspan="2">Categories Total</th><th class="col-total" rowspan="2">E.Total</th>
        <th class="col-deduction" colspan="3">M. Deduction</th>
        <th class="col-total" rowspan="2">Final Score</th>
      </tr>
      <tr>
        ${tevHeaders}${pevHeaders}
        <th class="col-deduction">Stop</th><th class="col-deduction">Discard</th><th class="col-deduction">Cut</th>
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
  th.col-tex { color: #7c3aed; background: rgba(124, 58, 237, 0.22); }
  th.col-tev { color: #db2777; background: rgba(219, 39, 119, 0.18); }
  th.col-pev { color: #1d4ed8; background: rgba(29, 78, 216, 0.22); }
  th.col-total { color: #b45309; background: rgba(180, 83, 9, 0.22); }
  th.col-deduction { color: #db2777; background: rgba(219, 39, 119, 0.18); }
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
  <p class="muted" style="margin-bottom: 22px">Open a contest to manage divisions, judges, players and scores.</p>

  <form class="new-contest" @submit.prevent="createContest">
    <span class="lbl">Create a contest</span>
    <input v-model="newName" type="text" placeholder="Contest name" />
    <input v-model.number="newYear" type="number" placeholder="Year" style="width: 100px" />
    <button class="primary" type="submit" :disabled="creating">Create</button>
  </form>

  <p v-if="store.loading" class="muted">Loading…</p>

  <div
    v-for="contest in store.contests"
    :key="contest.id"
    class="card contest-card"
    :class="{ 'is-open': isOpen(contest.id) }"
    :style="contest.hidden ? 'opacity: 0.55' : ''"
  >
    <div class="contest-head" @click="toggleOpen(contest.id)">
      <span class="caret"><Icon name="chevron-right" :size="16" /></span>
      <div class="info" v-if="editingContestId !== contest.id">
        <div class="title">
          <span>{{ contest.name }}</span>
          <span class="badge-year">{{ contest.year }}</span>
          <span v-if="contest.locked" class="plain-pill" title="Locked by head judge">Locked</span>
          <span v-if="contest.hidden" class="plain-pill" title="Hidden from the general contest list — only superadmins can see it">Hidden</span>
        </div>
        <div class="meta">
          {{ contest.divisions.length }} division{{ contest.divisions.length === 1 ? '' : 's' }}
          <template v-if="usersById[contest.headJudgeUserId]"> · Head judge: {{ judgeName(contest.headJudgeUserId) }}</template>
        </div>
      </div>
      <div class="row" v-else @click.stop>
        <input v-model="editName" type="text" placeholder="Contest name" style="width: 220px" />
        <input v-model.number="editYear" type="number" placeholder="Year" style="width: 100px" />
        <button class="primary" :disabled="savingEdit" @click="saveContestEdit(contest)">
          {{ savingEdit ? 'Saving…' : 'Save' }}
        </button>
        <button :disabled="savingEdit" @click="cancelEditContest">Cancel</button>
      </div>
      <div class="contest-head-actions" @click.stop>
        <button v-if="auth.user?.isSuperAdmin && editingContestId !== contest.id" @click="startEditContest(contest)">Edit</button>
        <button :disabled="downloading[contest.id]" @click="downloadContestResults(contest)">
          {{ downloading[contest.id] ? 'Preparing…' : 'Download Results' }}
        </button>
        <button
          v-if="contest.headJudgeUserId === userId"
          :disabled="lockToggling[contest.id]"
          :title="contest.locked ? 'Unlock this contest so scores and settings can be changed again.' : 'Lock this contest to freeze all scores and settings.'"
          @click="toggleLock(contest)"
        >
          {{ lockToggling[contest.id] ? 'Working…' : contest.locked ? 'Unlock' : 'Lock' }}
        </button>
        <button
          v-if="auth.user?.isSuperAdmin"
          :disabled="hidingToggling[contest.id]"
          :title="contest.hidden ? 'Show this contest in the general contest list again.' : 'Hide this contest from the general contest list.'"
          @click="toggleHidden(contest)"
        >
          {{ hidingToggling[contest.id] ? 'Working…' : contest.hidden ? 'Show' : 'Hide' }}
        </button>
        <RouterLink :to="{ name: 'contest-edit', params: { contestId: contest.id } }">
          <button class="primary">Open →</button>
        </RouterLink>
      </div>
    </div>

    <div class="contest-body">
      <p v-if="!contest.divisions.length" class="muted">No divisions yet - open the contest to add one.</p>
      <template v-else>
        <template v-for="division in contest.divisions" :key="division.id">
          <div v-for="stage in orderedStages(division.stages)" :key="division.id + ':' + stage" class="division-stage-block">
            <div>
              <div class="dsb-title">
                <span class="div-name">{{ division.name }}</span>
                <span class="pill-stage" :class="stage">{{ stageLabel(stage) }}</span>
              </div>
              <table v-if="topThree(division.id, stage).length" class="top3-mini">
                <thead>
                  <tr><th></th><th>Player</th><th class="score-hd">Final Score</th></tr>
                </thead>
                <tbody>
                  <tr v-for="r in topThree(division.id, stage)" :key="r.playerId">
                    <td class="rank-cell">
                      <span
                        class="rank-badge"
                        :class="{ gold: r.place === 1, silver: r.place === 2, bronze: r.place === 3 }"
                      >{{ r.place }}</span>
                    </td>
                    <td>{{ r.name }}</td>
                    <td class="score-cell">{{ r.finalScore.toFixed(2) }}</td>
                  </tr>
                </tbody>
              </table>
              <p v-else class="no-results">No results yet.</p>
            </div>
            <div>
              <RouterLink :to="{ name: 'results', params: { contestId: contest.id, divisionId: division.id, stage } }">
                <button :disabled="!topThree(division.id, stage).length">Result Detail</button>
              </RouterLink>
            </div>
          </div>
        </template>
      </template>
    </div>
  </div>

  <p v-if="!store.loading && !store.contests.length" class="muted">
    No contests yet. Create one above, or ask a head judge to invite you.
  </p>
</template>
