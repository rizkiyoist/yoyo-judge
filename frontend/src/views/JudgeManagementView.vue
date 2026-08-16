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

const isOwner = computed(() => contest.value?.ownerUserId === auth.user?.id)
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
  const allSearched = await api.searchUsers('example.com')
  usersById.value = Object.fromEntries(allSearched.map((u) => [u.id, u]))
}

onMounted(load)

async function search() {
  searchResults.value = query.value.trim() ? await api.searchUsers(query.value) : []
}

// Invites always land on whichever stage tab is currently active.
async function invite(user: User) {
  if (!selectedDivisionId.value) return
  await api.inviteJudge(props.contestId, selectedDivisionId.value, selectedStage.value, user.id, selectedRole.value, selectedSlot.value)
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
  return u ? `${u.firstName} ${u.lastName} (${u.email})` : userId
}

function isHeadJudge(userId: string): boolean {
  return userId === contest.value?.ownerUserId
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
    <h1>{{ contest.name }} — Judges</h1>
    <p v-if="!isOwner" class="muted">Only the head judge who created this contest can invite judges.</p>

    <div v-if="isOwner" class="card">
      <h2>Invite a judge</h2>
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
        <label>Search judges by name or email</label>
        <input v-model="query" type="text" placeholder="e.g. clicker.a1@example.com" @input="search" />
      </div>

      <div v-if="searchResults.length" class="row" style="flex-direction: column; align-items: stretch">
        <button v-for="u in searchResults" :key="u.id" @click="invite(u)">
          Invite {{ u.firstName }} {{ u.lastName }} — {{ u.email }}
        </button>
      </div>
    </div>

    <div class="card">
      <h2 style="margin-bottom: 10px">Current assignments — {{ selectedDivision?.name }}</h2>
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
                <th v-if="isOwner"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in clickerAssignments" :key="a.id">
                <td>{{ a.slot }}</td>
                <td>
                  <strong v-if="isHeadJudge(a.userId)">{{ userLabel(a.userId) }}</strong>
                  <template v-else>{{ userLabel(a.userId) }}</template>
                </td>
                <td v-if="isOwner"><button class="danger" @click="remove(a)">Remove</button></td>
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
                <th v-if="isOwner"></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="a in evalAssignments" :key="a.id">
                <td>{{ a.slot }}</td>
                <td>
                  <strong v-if="isHeadJudge(a.userId)">{{ userLabel(a.userId) }}</strong>
                  <template v-else>{{ userLabel(a.userId) }}</template>
                </td>
                <td v-if="isOwner"><button class="danger" @click="remove(a)">Remove</button></td>
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
