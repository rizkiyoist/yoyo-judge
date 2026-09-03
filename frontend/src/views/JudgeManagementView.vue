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
// Stage(s) to invite into - 'both' invites the same slot into every stage
// the division has; slot itself is auto-assigned, not picked.
const inviteStage = ref<'both' | ScoringStage>('both')
const inviteError = ref('')

const isOwnerUser = computed(() => contest.value?.ownerUserId === auth.user?.id)
const isCurrentHeadJudge = computed(() => contest.value?.headJudgeUserId === auth.user?.id)
// Inviting/removing judges, adding/removing divisions & players, and
// transferring head-judge status are shared by the owner and the current
// head judge; locking/unlocking is head-judge-only (see role table in
// docs/PROGRESS.md).
const canManage = computed(() => isOwnerUser.value || isCurrentHeadJudge.value)
const hasDivisions = computed(() => (contest.value?.divisions.length ?? 0) > 0)
const selectedDivision = computed(() => contest.value?.divisions.find((d) => d.id === selectedDivisionId.value))

const inviteStageOptions = computed(() => {
  const stages = selectedDivision.value?.stages ?? []
  const opts: { value: 'both' | ScoringStage; label: string }[] = []
  if (stages.length > 1) opts.push({ value: 'both', label: 'Both' })
  if (stages.includes('prelim')) opts.push({ value: 'prelim', label: 'Prelim' })
  if (stages.includes('final')) opts.push({ value: 'final', label: 'Final' })
  return opts
})

// Which stages an invite actually targets, given the current choice.
const inviteTargetStages = computed<ScoringStage[]>(() => {
  const stages = selectedDivision.value?.stages ?? []
  if (inviteStage.value === 'both') return stages
  return stages.includes(inviteStage.value) ? [inviteStage.value] : stages
})

// Lowest slot (1-6) free in every targeted stage for the selected role, so
// the same judge gets the same slot number across prelim and final. null
// if all 6 slots are already taken in at least one targeted stage.
const nextInviteSlot = computed<number | null>(() => {
  const stages = inviteTargetStages.value
  if (!stages.length) return null
  for (let slot = 1; slot <= 6; slot++) {
    const taken = stages.some((stage) =>
      assignments.value.some(
        (a) =>
          a.divisionId === selectedDivisionId.value &&
          a.stage === stage &&
          a.role === selectedRole.value &&
          a.slot === slot,
      ),
    )
    if (!taken) return slot
  }
  return null
})

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
    inviteStage.value = division && division.stages.length === 1 ? division.stages[0] : 'both'
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

async function invite(identity: { userId: string } | { email: string }) {
  if (!selectedDivisionId.value) return
  const stages = inviteTargetStages.value
  const slot = nextInviteSlot.value
  if (!stages.length || slot === null) {
    inviteError.value = 'All slots for this role are already assigned.'
    return
  }
  inviteError.value = ''
  for (const stage of stages) {
    await api.inviteJudge(props.contestId, selectedDivisionId.value, stage, identity, selectedRole.value, slot)
  }
  query.value = ''
  searchResults.value = []
  await load()
}

const removing = ref<Record<string, boolean>>({})
const removeError = ref<Record<string, string>>({})

async function remove(assignment: JudgeAssignment) {
  removing.value[assignment.id] = true
  removeError.value[assignment.id] = ''
  try {
    await api.removeJudgeAssignment(props.contestId, assignment.id)
    await load()
  } catch (e) {
    removeError.value[assignment.id] = e instanceof Error ? e.message : 'Failed to remove judge'
  } finally {
    removing.value[assignment.id] = false
  }
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

// Major deduction is always an additional role on top of an existing
// clicker/evaluator assignment for the same division+stage - never
// assigned to someone with no other role there.
const mdAssignment = computed(() => assignmentsForSelection.value.find((a) => a.role === 'major_deduction'))
const mdCandidates = computed(() => {
  const ids = new Set(
    assignmentsForSelection.value.filter((a) => a.role !== 'major_deduction').map((a) => a.userId),
  )
  return [...ids]
})
const mdTargetId = ref('')
const assigningMd = ref(false)
const mdError = ref('')

async function assignMd() {
  if (!selectedDivisionId.value || !mdTargetId.value) return
  assigningMd.value = true
  mdError.value = ''
  try {
    await api.inviteJudge(
      props.contestId,
      selectedDivisionId.value,
      selectedStage.value,
      { userId: mdTargetId.value },
      'major_deduction',
      1,
    )
    mdTargetId.value = ''
    await load()
  } catch (e) {
    mdError.value = e instanceof Error ? e.message : 'Failed to assign major deduction judge'
  } finally {
    assigningMd.value = false
  }
}
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <div class="row" style="justify-content: space-between; align-items: center">
      <h1 style="margin: 0">{{ contest.name }} - Judges</h1>
      <div v-if="hasDivisions" class="row" style="gap: 4px">
        <button
          v-for="d in contest.divisions"
          :key="d.id"
          :class="{ primary: selectedDivisionId === d.id }"
          @click="selectedDivisionId = d.id"
        >
          {{ d.name }}
        </button>
      </div>
    </div>
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
      <p class="muted" style="margin-top: -6px">
        Inviting into <strong>{{ selectedDivision?.name }}</strong> - pick a different division from the buttons above to change this.
      </p>
      <div class="row">
        <div class="field">
          <label>Role</label>
          <select v-model="selectedRole">
            <option value="clicker">Clicker Judge (TEx)</option>
            <option value="evaluator">Evaluation Judge (PEv)</option>
          </select>
        </div>
        <div class="field">
          <label>Stage</label>
          <select v-model="inviteStage">
            <option v-for="opt in inviteStageOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
          </select>
        </div>
      </div>
      <p class="muted" style="margin-top: -6px">
        <template v-if="nextInviteSlot !== null">Will be assigned slot {{ nextInviteSlot }}.</template>
        <template v-else>All slots for this role are already assigned.</template>
      </p>

      <div class="field">
        <label>Search judges by name, or invite by email</label>
        <input v-model="query" type="text" placeholder="e.g. clicker.a1@example.com" @input="search" />
      </div>

      <div v-if="searchResults.length" class="row" style="flex-direction: column; align-items: stretch">
        <button v-for="u in searchResults" :key="u.id" :disabled="nextInviteSlot === null" @click="invite({ userId: u.id })">
          Invite {{ u.firstName }} {{ u.lastName }} - {{ u.email }}
        </button>
      </div>
      <div v-else-if="canInviteByEmail" class="row" style="flex-direction: column; align-items: stretch">
        <button :disabled="nextInviteSlot === null" @click="invite({ email: query.trim() })">
          Invite {{ query.trim() }} (not registered yet)
        </button>
        <p class="muted" style="font-size: 0.85em; margin-top: 4px">
          No account with this email yet - inviting reserves their slot, and they'll see this
          contest as soon as they sign in with this email.
        </p>
      </div>
      <p v-if="inviteError" class="error">{{ inviteError }}</p>
      </fieldset>
    </div>

    <div v-if="hasDivisions" class="card">
      <h2 style="margin-bottom: 10px">Current assignments</h2>
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
                  <div v-if="removeError[a.id]" class="error" style="font-size: 0.85em">{{ removeError[a.id] }}</div>
                </td>
                <td v-if="canManage">
                  <button class="danger" :disabled="contest.locked || removing[a.id]" @click="remove(a)">
                    {{ removing[a.id] ? 'Removing…' : 'Remove' }}
                  </button>
                </td>
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
                  <div v-if="removeError[a.id]" class="error" style="font-size: 0.85em">{{ removeError[a.id] }}</div>
                </td>
                <td v-if="canManage">
                  <button class="danger" :disabled="contest.locked || removing[a.id]" @click="remove(a)">
                    {{ removing[a.id] ? 'Removing…' : 'Remove' }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else class="muted">None assigned.</p>
        </div>
      </div>

      <div style="margin-top: 24px">
        <h3 style="margin: 0 0 6px">Major Deduction Judge</h3>
        <p class="muted" style="font-size: 0.85em; margin-top: 0">
          Always an additional role for a judge already assigned above.
        </p>
        <table v-if="mdAssignment" style="max-width: 400px">
          <thead>
            <tr>
              <th>Judge</th>
              <th v-if="canManage"></th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>
                <strong v-if="isOwnerUserId(mdAssignment.userId)">{{ userLabel(mdAssignment.userId) }}</strong>
                <template v-else>{{ userLabel(mdAssignment.userId) }}</template>
                <span v-if="isHeadJudgeUserId(mdAssignment.userId)" class="badge">Head Judge</span>
                <div v-if="removeError[mdAssignment.id]" class="error" style="font-size: 0.85em">
                  {{ removeError[mdAssignment.id] }}
                </div>
              </td>
              <td v-if="canManage">
                <button class="danger" :disabled="contest.locked || removing[mdAssignment.id]" @click="remove(mdAssignment)">
                  {{ removing[mdAssignment.id] ? 'Removing…' : 'Remove' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        <template v-else>
          <p class="muted">None assigned.</p>
          <div v-if="canManage" class="row" style="margin-top: 8px">
            <select v-model="mdTargetId" :disabled="contest.locked || !mdCandidates.length">
              <option value="" disabled>Choose a judge…</option>
              <option v-for="uid in mdCandidates" :key="uid" :value="uid">{{ userLabel(uid) }}</option>
            </select>
            <button :disabled="!mdTargetId || contest.locked || assigningMd" @click="assignMd">
              {{ assigningMd ? 'Working…' : 'Assign' }}
            </button>
          </div>
          <p v-if="canManage && !mdCandidates.length" class="muted" style="font-size: 0.85em">
            Invite a clicker or evaluation judge first.
          </p>
        </template>
        <span v-if="mdError" class="error">{{ mdError }}</span>
      </div>
    </div>

    <div v-if="canManage" class="card">
      <h2>Head judge</h2>
      <p class="muted">
        The head judge can override any judge's scores and lock/unlock the contest.
      </p>
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
  </div>
  <p v-else class="muted">Loading…</p>
</template>
