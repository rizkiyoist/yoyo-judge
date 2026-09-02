<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import type { Contest, JudgeAssignment, JudgeRole, ScoringStage, User } from '../types'

const props = defineProps<{ contestId: string }>()

const auth = useAuthStore()
const contest = ref<Contest | null>(null)
const assignments = ref<JudgeAssignment[]>([])
const usersById = ref<Record<string, User>>({})

const query = ref('')
const searchResults = ref<User[]>([])
const selectedDivisionId = ref('')
const selectedStage = ref<ScoringStage>('final')
const selectedRole = ref<JudgeRole>('clicker')
const selectedSlot = ref(1)

const isOwnerUser = computed(() => contest.value?.ownerUserId === auth.user?.id)
const isCurrentHeadJudge = computed(() => contest.value?.headJudgeUserId === auth.user?.id)
// Inviting/removing judges, adding/removing divisions & players, and
// transferring head-judge status are shared by the owner and the current
// head judge; locking/unlocking is head-judge-only (see role table in
// docs/PROGRESS.md).
const canManage = computed(() => isOwnerUser.value || isCurrentHeadJudge.value)
const hasDivisions = computed(() => (contest.value?.divisions.length ?? 0) > 0)
const selectedDivision = computed(() => contest.value?.divisions.find((d) => d.id === selectedDivisionId.value))

function stageLabel(stage: ScoringStage): string {
  return stage === 'prelim' ? 'Prelim' : 'Final'
}

// Always keep a division+stage selected once one is available, so
// inviting/searching works immediately without requiring a manual pick.
watch(
  () => contest.value?.divisions,
  (divisions) => {
    if (!divisions?.length) return
    if (!divisions.some((d) => d.id === selectedDivisionId.value)) {
      selectedDivisionId.value = divisions[0].id
    }
  },
  { immediate: true },
)
watch(
  selectedDivision,
  (division) => {
    if (division && !division.stages.includes(selectedStage.value)) {
      selectedStage.value = division.stages[0] ?? 'final'
    }
  },
  { immediate: true },
)

async function load() {
  contest.value = await api.getContest(props.contestId)
  assignments.value = await api.listJudgeAssignments(props.contestId)
  const ids = new Set(assignments.value.map((a) => a.userId))
  if (contest.value) ids.add(contest.value.ownerUserId)
  const users = await api.getUsers([...ids])
  usersById.value = Object.fromEntries(users.map((u) => [u.id, u]))
}

onMounted(load)

async function search() {
  searchResults.value = query.value.trim() ? await api.searchUsers(query.value) : []
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
// Only offered once the search comes back empty for a query that's a
// plausible email - avoids suggesting it while results are still loading
// or for a plain name search with no matches.
const canInviteByEmail = computed(() => !searchResults.value.length && EMAIL_RE.test(query.value.trim()))

// Invites always land on whichever stage tab is currently active.
async function invite(identity: { userId: string } | { email: string }) {
  if (!selectedDivisionId.value) return
  await api.inviteJudge(
    props.contestId,
    selectedDivisionId.value,
    selectedStage.value,
    identity,
    selectedRole.value,
    selectedSlot.value,
  )
  query.value = ''
  searchResults.value = []
  await load()
}

async function remove(assignment: JudgeAssignment) {
  await api.removeJudgeAssignment(props.contestId, assignment.id)
  await load()
}

function userLabel(userId: string): string {
  const u = usersById.value[userId]
  if (!u) return userId
  const name = `${u.firstName} ${u.lastName}`.trim()
  return name ? `${name} (${u.email})` : `${u.email} (not signed in yet)`
}

function isOwnerUserId(userId: string): boolean {
  return userId === contest.value?.ownerUserId
}

function isHeadJudgeUserId(userId: string): boolean {
  return userId === contest.value?.headJudgeUserId
}

// Anyone eligible to receive head-judge status: the owner, plus every
// distinct judge currently assigned somewhere in this contest.
const headJudgeCandidates = computed(() => {
  if (!contest.value) return []
  const ids = new Set(assignments.value.map((a) => a.userId))
  ids.add(contest.value.ownerUserId)
  return [...ids]
})
const transferTargetId = ref('')
const transferring = ref(false)
const transferError = ref('')

async function transferHeadJudge() {
  if (!contest.value || !transferTargetId.value) return
  transferring.value = true
  transferError.value = ''
  try {
    await api.transferHeadJudge(contest.value.id, transferTargetId.value)
    await load()
  } catch (e) {
    transferError.value = e instanceof Error ? e.message : 'Failed to transfer head judge'
  } finally {
    transferring.value = false
  }
}

const assignmentsForSelection = computed(() =>
  assignments.value.filter((a) => a.divisionId === selectedDivisionId.value && a.stage === selectedStage.value),
)
const clickerAssignments = computed(() =>
  assignmentsForSelection.value.filter((a) => a.role === 'clicker').sort((a, b) => a.slot - b.slot),
)
const evalAssignments = computed(() =>
  assignmentsForSelection.value.filter((a) => a.role === 'evaluator').sort((a, b) => a.slot - b.slot),
)
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <h1>{{ contest.name }} - Judges</h1>
    <p v-if="!canManage" class="muted">Only the contest owner or head judge can manage judges.</p>
    <p v-if="contest.locked" class="error">
      🔒 This contest is locked. {{ isCurrentHeadJudge ? 'Unlock it below to make changes.' : 'Only the head judge can unlock it.' }}
    </p>

    <div v-if="canManage && !hasDivisions" class="card">
      <p class="error">
        You can add judges after setting divisions.
        <RouterLink :to="{ name: 'contest-edit', params: { contestId } }">Go to Divisions</RouterLink>
      </p>
    </div>

    <div v-if="canManage" class="card" :style="!hasDivisions ? 'opacity: 0.5; pointer-events: none' : ''">
      <h2>Invite a judge</h2>
      <fieldset :disabled="contest.locked || !hasDivisions" style="border: 0; padding: 0; margin: 0">
      <div class="row">
        <div class="field">
          <label>Division</label>
          <select v-model="selectedDivisionId">
            <option v-for="d in contest.divisions" :key="d.id" :value="d.id">{{ d.name }}</option>
          </select>
        </div>
        <div class="field">
          <label>Role</label>
          <select v-model="selectedRole">
            <option value="clicker">Clicker Judge (TEx)</option>
            <option value="evaluator">Evaluation Judge (PEv)</option>
          </select>
        </div>
        <div class="field">
          <label>Slot</label>
          <select v-model.number="selectedSlot">
            <option v-for="n in 6" :key="n" :value="n">{{ n }}</option>
          </select>
        </div>
      </div>
      <p class="muted" style="margin-top: -6px">
        Invites go to the <strong>{{ stageLabel(selectedStage) }}</strong> tab selected below.
      </p>

      <div class="field">
        <label>Search judges by name, or invite by email</label>
        <input v-model="query" type="text" placeholder="e.g. clicker.a1@example.com" @input="search" />
      </div>

      <div v-if="searchResults.length" class="row" style="flex-direction: column; align-items: stretch">
        <button v-for="u in searchResults" :key="u.id" @click="invite({ userId: u.id })">
          Invite {{ u.firstName }} {{ u.lastName }} - {{ u.email }}
        </button>
      </div>
      <div v-else-if="canInviteByEmail" class="row" style="flex-direction: column; align-items: stretch">
        <button @click="invite({ email: query.trim() })">Invite {{ query.trim() }} (not registered yet)</button>
        <p class="muted" style="font-size: 0.85em; margin-top: 4px">
          No account with this email yet - inviting reserves their slot, and they'll see this
          contest as soon as they sign in with this email.
        </p>
      </div>
      </fieldset>
    </div>

    <div v-if="canManage" class="card">
      <h2>Head judge</h2>
      <p class="muted">
        Currently: <strong>{{ userLabel(contest.headJudgeUserId) }}</strong>.
        Can be handed to any invited judge.
      </p>
      <div class="row">
        <select v-model="transferTargetId" :disabled="contest.locked">
          <option value="" disabled>Choose a judge…</option>
          <option v-for="uid in headJudgeCandidates" :key="uid" :value="uid">{{ userLabel(uid) }}</option>
        </select>
        <button :disabled="!transferTargetId || contest.locked || transferring" @click="transferHeadJudge">
          {{ transferring ? 'Working…' : 'Make head judge' }}
        </button>
      </div>
      <span v-if="transferError" class="error">{{ transferError }}</span>
    </div>

    <div v-if="hasDivisions" class="card">
      <h2 style="margin-bottom: 10px">Current assignments</h2>
      <p class="muted" style="margin-top: -4px; margin-bottom: 12px">Division: <strong>{{ selectedDivision?.name }}</strong></p>
      <div class="row" style="gap: 4px; margin-bottom: 14px">
        <button
          v-for="s in selectedDivision?.stages ?? []"
          :key="s"
          :class="{ primary: selectedStage === s }"
          @click="selectedStage = s"
        >
          {{ stageLabel(s) }}
        </button>
      </div>

      <div class="row" style="align-items: flex-start; gap: 32px">
        <div style="flex: 1">
          <h3 style="margin: 0 0 6px">Clicker Judges (TEx)</h3>
          <table v-if="clickerAssignments.length">
            <thead>
              <tr>
                <th>Slot</th>
                <th>Judge</th>
                <th v-if="canManage"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in clickerAssignments" :key="a.id">
                <td>{{ a.slot }}</td>
                <td>
                  <strong v-if="isOwnerUserId(a.userId)">{{ userLabel(a.userId) }}</strong>
                  <template v-else>{{ userLabel(a.userId) }}</template>
                  <span v-if="isHeadJudgeUserId(a.userId)" class="badge">Head Judge</span>
                </td>
                <td v-if="canManage"><button class="danger" :disabled="contest.locked" @click="remove(a)">Remove</button></td>
              </tr>
            </tbody>
          </table>
          <p v-else class="muted">None assigned.</p>
        </div>

        <div style="flex: 1">
          <h3 style="margin: 0 0 6px">Evaluation Judges (PEv)</h3>
          <table v-if="evalAssignments.length">
            <thead>
              <tr>
                <th>Slot</th>
                <th>Judge</th>
                <th v-if="canManage"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in evalAssignments" :key="a.id">
                <td>{{ a.slot }}</td>
                <td>
                  <strong v-if="isOwnerUserId(a.userId)">{{ userLabel(a.userId) }}</strong>
                  <template v-else>{{ userLabel(a.userId) }}</template>
                  <span v-if="isHeadJudgeUserId(a.userId)" class="badge">Head Judge</span>
                </td>
                <td v-if="canManage"><button class="danger" :disabled="contest.locked" @click="remove(a)">Remove</button></td>
              </tr>
            </tbody>
          </table>
          <p v-else class="muted">None assigned.</p>
        </div>
      </div>
    </div>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
