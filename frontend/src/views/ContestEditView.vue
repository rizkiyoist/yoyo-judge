<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import { useContestStore } from '../stores/contests'
import type { Contest, ScoringStage } from '../types'

const props = defineProps<{ contestId: string }>()

const auth = useAuthStore()
const store = useContestStore()
const contest = ref<Contest | null>(null)

const divisionName = ref('')
const wantsPrelim = ref(true)
const wantsFinal = ref(true)
const saving = ref(false)

const isOwner = computed(() => contest.value?.ownerUserId === auth.user?.id)

async function load() {
  contest.value = await api.getContest(props.contestId)
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
</script>

<template>
  <div v-if="contest">
    <h1>{{ contest.name }} — Divisions</h1>

    <p v-if="!isOwner" class="muted">Only the head judge who created this contest can edit divisions.</p>

    <div v-if="isOwner" class="card">
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
          </tr>
        </thead>
        <tbody>
          <tr v-for="division in contest.divisions" :key="division.id">
            <td>{{ division.name }}</td>
            <td>
              <input
                type="checkbox"
                :checked="division.stages.includes('prelim')"
                :disabled="!isOwner"
                @change="toggleStage(division.id, division.stages, 'prelim')"
              />
            </td>
            <td>
              <input
                type="checkbox"
                :checked="division.stages.includes('final')"
                :disabled="!isOwner"
                @change="toggleStage(division.id, division.stages, 'final')"
              />
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="muted">No divisions yet.</p>
    </div>
  </div>
  <p v-else class="muted">Loading…</p>
</template>
