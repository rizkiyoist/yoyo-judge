<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useContestStore } from '../stores/contests'

const auth = useAuthStore()
const store = useContestStore()
const newName = ref('')
const creating = ref(false)

const userId = computed(() => auth.user?.id ?? '')

onMounted(async () => {
  if (userId.value) await store.fetchContests(userId.value)
})

async function createContest() {
  if (!newName.value.trim()) return
  creating.value = true
  try {
    await store.createContest(newName.value.trim(), userId.value)
    newName.value = ''
  } finally {
    creating.value = false
  }
}
</script>

<template>
  <h1>Contests</h1>

  <div class="card">
    <h2>Create a contest</h2>
    <form class="row" @submit.prevent="createContest">
      <input v-model="newName" type="text" placeholder="Contest name" style="flex: 1" />
      <button class="primary" type="submit" :disabled="creating">Create</button>
    </form>
  </div>

  <p v-if="store.loading" class="muted">Loading…</p>

  <div v-for="contest in store.contests" :key="contest.id" class="card">
    <div class="row" style="justify-content: space-between">
      <h2>{{ contest.name }}</h2>
      <div class="row">
        <RouterLink :to="{ name: 'contest-judges', params: { contestId: contest.id } }">
          <button>Judges</button>
        </RouterLink>
        <RouterLink :to="{ name: 'contest-edit', params: { contestId: contest.id } }">
          <button>Divisions</button>
        </RouterLink>
      </div>
    </div>

    <p v-if="!contest.divisions.length" class="muted">No divisions yet — add one to get started.</p>

    <table v-else>
      <thead>
        <tr>
          <th>Division</th>
          <th>Stages</th>
          <th></th>
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
            <div class="row">
              <RouterLink :to="{ name: 'division-players', params: { contestId: contest.id, divisionId: division.id } }">
                Players
              </RouterLink>
              <RouterLink
                v-for="stage in division.stages"
                :key="stage"
                :to="{ name: 'score-entry', params: { contestId: contest.id, divisionId: division.id, stage } }"
              >
                Score ({{ stage }})
              </RouterLink>
              <RouterLink
                v-for="stage in division.stages"
                :key="stage + '-results'"
                :to="{ name: 'results', params: { contestId: contest.id, divisionId: division.id, stage } }"
              >
                Results ({{ stage }})
              </RouterLink>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>

  <p v-if="!store.loading && !store.contests.length" class="muted">
    No contests yet. Create one above, or ask a head judge to invite you.
  </p>
</template>
