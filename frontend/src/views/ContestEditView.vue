<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import { useContestStore } from '../stores/contests'
import type { Contest, Player, ScoringStage } from '../types'

const props = defineProps<{ contestId: string }>()

const auth = useAuthStore()
const store = useContestStore()
const contest = ref<Contest | null>(null)
const playersByDivision = ref<Record<string, Player[]>>({})

const divisionName = ref('')
const wantsPrelim = ref(true)
const wantsFinal = ref(true)
const saving = ref(false)
const deleting = ref<Record<string, boolean>>({})
const deleteError = ref<Record<string, string>>({})

const canEdit = computed(
  () => !!auth.user && (contest.value?.ownerUserId === auth.user.id || contest.value?.headJudgeUserId === auth.user.id),
)

async function load() {
  contest.value = await api.getContest(props.contestId)
  if (!contest.value) return
  const entries = await Promise.all(
    contest.value.divisions.map(async (d) => [d.id, await api.listPlayers(d.id)] as const),
  )
  playersByDivision.value = Object.fromEntries(entries)
}

onMounted(load)

function selectedStages(): ScoringStage[] {
  const stages: ScoringStage[] = []
  if (wantsPrelim.value) stages.push('prelim')
  if (wantsFinal.value) stages.push('final')
  return stages
}

async function addDivision() {
  if (!divisionName.value.trim() || !contest.value) return
  const stages = selectedStages()
  if (!stages.length) return
  saving.value = true
  try {
    await store.addDivision(contest.value.id, divisionName.value.trim(), stages)
    divisionName.value = ''
    await load()
  } finally {
    saving.value = false
  }
}

async function toggleStage(divisionId: string, currentStages: ScoringStage[], stage: ScoringStage) {
  if (!contest.value) return
  const next = currentStages.includes(stage)
    ? currentStages.filter((s) => s !== stage)
    : [...currentStages, stage]
  await store.updateDivisionStages(contest.value.id, divisionId, next)
  await load()
}

function playerCount(divisionId: string): number {
  return playersByDivision.value[divisionId]?.length ?? 0
}

async function deleteDivision(divisionId: string) {
  if (!contest.value) return
  if (!confirm('Delete this division? This cannot be undone.')) return
  deleting.value[divisionId] = true
  deleteError.value[divisionId] = ''
  try {
    await store.deleteDivision(contest.value.id, divisionId)
    await load()
  } catch (e) {
    deleteError.value[divisionId] = e instanceof Error ? e.message : 'Failed to delete division'
  } finally {
    deleting.value[divisionId] = false
  }
}
</script>

<template>
  <div v-if="contest">
    <RouterLink :to="{ name: 'contests' }">&larr; Back to contests</RouterLink>
    <div class="row" style="justify-content: space-between; align-items: center">
      <h1 style="margin: 0">{{ contest.name }} - Divisions</h1>
      <RouterLink :to="{ name: 'contest-judges', params: { contestId } }">
        <button>Go to Judges</button>
      </RouterLink>
    </div>

    <p v-if="!canEdit" class="muted">Only the contest owner or head judge can edit divisions.</p>
    <p v-if="contest.locked" class="error">
      🔒 This contest is locked - unlock it from the Judges page before editing divisions.
    </p>

    <div v-if="canEdit && !contest.locked" class="card">
      <h2>Add a division</h2>
      <div class="field">
        <label>Division name</label>
        <input v-model="divisionName" type="text" placeholder="e.g. 3A" />
      </div>
      <div class="row">
        <label><input v-model="wantsPrelim" type="checkbox" /> Prelim</label>
        <label><input v-model="wantsFinal" type="checkbox" /> Final</label>
      </div>
      <button class="primary" style="margin-top: 10px" :disabled="saving" @click="addDivision">Add division</button>
    </div>

    <div class="card">
      <h2>Divisions</h2>
      <table v-if="contest.divisions.length">
        <thead>
          <tr>
            <th>Name</th>
            <th>Prelim</th>
            <th>Final</th>
            <th>Players</th>
            <th v-if="canEdit"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="division in contest.divisions" :key="division.id">
            <td>{{ division.name }}</td>
            <td>
              <input
                type="checkbox"
                :checked="division.stages.includes('prelim')"
                :disabled="!canEdit || contest.locked"
                @change="toggleStage(division.id, division.stages, 'prelim')"
              />
            </td>
            <td>
              <input
                type="checkbox"
                :checked="division.stages.includes('final')"
                :disabled="!canEdit || contest.locked"
                @change="toggleStage(division.id, division.stages, 'final')"
              />
            </td>
            <td>{{ playerCount(division.id) }}</td>
            <td v-if="canEdit">
              <button
                class="danger"
                :disabled="contest.locked || deleting[division.id] || playerCount(division.id) > 0"
                :title="playerCount(division.id) > 0 ? 'Remove all players from this division first' : ''"
                @click="deleteDivision(division.id)"
              >
                {{ deleting[division.id] ? 'Deleting…' : 'Delete' }}
              </button>
              <div v-if="deleteError[division.id]" class="error" style="font-size: 0.85em">
                {{ deleteError[division.id] }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="muted">No divisions yet.</p>
    </div>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
