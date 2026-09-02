<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import type { Contest, Player } from '../types'

const props = defineProps<{ contestId: string; divisionId: string }>()

const auth = useAuthStore()
const contest = ref<Contest | null>(null)
const players = ref<Player[]>([])

const nextNumber = ref(1)
const name = ref('')
const saving = ref(false)
const removing = ref<Record<string, boolean>>({})

const canEdit = computed(
  () => !!auth.user && (contest.value?.ownerUserId === auth.user.id || contest.value?.headJudgeUserId === auth.user.id),
)
const division = computed(() => contest.value?.divisions.find((d) => d.id === props.divisionId))

async function load() {
  contest.value = await api.getContest(props.contestId)
  players.value = await api.listPlayers(props.divisionId)
  nextNumber.value = (players.value.at(-1)?.number ?? 0) + 1
}

onMounted(load)

async function addPlayer() {
  if (!name.value.trim()) return
  saving.value = true
  try {
    await api.addPlayer(props.divisionId, nextNumber.value, name.value.trim())
    name.value = ''
    await load()
  } finally {
    saving.value = false
  }
}

async function removePlayer(playerId: string) {
  if (!confirm('Remove this player? Their scores for this division will be deleted too.')) return
  removing.value[playerId] = true
  try {
    await api.removePlayer(props.divisionId, playerId)
    await load()
  } finally {
    removing.value[playerId] = false
  }
}
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <h1>{{ contest.name }} - {{ division?.name }} players</h1>

    <p v-if="contest.locked" class="error">
      🔒 This contest is locked - unlock it from the Judges page before changing the roster.
    </p>

    <div v-if="canEdit && !contest.locked" class="card">
      <h2>Add a player</h2>
      <div class="row">
        <span class="badge">#{{ nextNumber }}</span>
        <div class="field" style="flex: 1">
          <label>Name</label>
          <input v-model="name" type="text" placeholder="Player name" @keyup.enter="addPlayer" />
        </div>
      </div>
      <button class="primary" :disabled="saving" @click="addPlayer">Add player</button>
    </div>

    <div class="card">
      <h2>Roster</h2>
      <table v-if="players.length">
        <thead>
          <tr>
            <th>#</th>
            <th>Name</th>
            <th v-if="canEdit"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in players" :key="p.id">
            <td>{{ p.number }}</td>
            <td>{{ p.name }}</td>
            <td v-if="canEdit">
              <button class="danger" :disabled="contest.locked || removing[p.id]" @click="removePlayer(p.id)">
                {{ removing[p.id] ? 'Removing…' : 'Remove' }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="muted">No players yet.</p>
    </div>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
