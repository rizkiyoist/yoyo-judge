<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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

async function load() {
  contest.value = await api.getContest(props.contestId)
  assignments.value = await api.listJudgeAssignments(props.contestId)
  if (contest.value?.divisions.length && !selectedDivisionId.value) {
    selectedDivisionId.value = contest.value.divisions[0].id
    selectedStage.value = contest.value.divisions[0].stages[0] ?? 'final'
  }
  const allSearched = await api.searchUsers('example.com')
  usersById.value = Object.fromEntries(allSearched.map((u) => [u.id, u]))
}

onMounted(load)

async function search() {
  searchResults.value = query.value.trim() ? await api.searchUsers(query.value) : []
}

async function invite(user: User) {
  if (!selectedDivisionId.value) return
  await api.inviteJudge(props.contestId, selectedDivisionId.value, selectedStage.value, user.id, selectedRole.value, selectedSlot.value)
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

const assignmentsForSelection = computed(() =>
  assignments.value.filter((a) => a.divisionId === selectedDivisionId.value && a.stage === selectedStage.value),
)
</script>

<template>
  <div v-if="contest">
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
          <label>Stage</label>
          <select v-model="selectedStage">
            <option v-for="s in selectedDivision?.stages ?? []" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
        <div class="field">
          <label>Role</label>
          <select v-model="selectedRole">
            <option value="clicker">Clicker judge (J-A)</option>
            <option value="evaluator">Evaluation judge (J-B)</option>
          </select>
        </div>
        <div class="field">
          <label>Slot</label>
          <select v-model.number="selectedSlot">
            <option v-for="n in 6" :key="n" :value="n">{{ n }}</option>
          </select>
        </div>
      </div>

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
      <h2>Current assignments — {{ selectedDivision?.name }} ({{ selectedStage }})</h2>
      <table v-if="assignmentsForSelection.length">
        <thead>
          <tr>
            <th>Role</th>
            <th>Slot</th>
            <th>Judge</th>
            <th v-if="isOwner"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in assignmentsForSelection" :key="a.id">
            <td>{{ a.role === 'clicker' ? 'Clicker (J-A)' : 'Evaluator (J-B)' }}</td>
            <td>{{ a.slot }}</td>
            <td>{{ userLabel(a.userId) }}</td>
            <td v-if="isOwner"><button class="danger" @click="remove(a)">Remove</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="muted">No judges assigned yet.</p>
    </div>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
