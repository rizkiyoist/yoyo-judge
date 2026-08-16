<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import { useContestStore } from '../stores/contests'
import type { JudgeAssignment, User } from '../types'

const auth = useAuthStore()
const store = useContestStore()
const newName = ref('')
const creating = ref(false)

const userId = computed(() => auth.user?.id ?? '')
const judgesByContest = ref<Record<string, JudgeAssignment[]>>({})
const usersById = ref<Record<string, User>>({})

onMounted(async () => {
  if (userId.value) await store.fetchContests(userId.value)
})

// Load each contest's judge assignments (to show "Judges: ...") once the
// contest list is known, and re-run whenever it changes (e.g. after create).
watch(
  () => store.contests,
  async (contests) => {
    const allUsers = await api.searchUsers('example.com')
    usersById.value = Object.fromEntries(allUsers.map((u) => [u.id, u]))
    for (const contest of contests) {
      judgesByContest.value[contest.id] = await api.listJudgeAssignments(contest.id)
    }
  },
  { deep: false },
)

function judgeName(userId: string): string {
  const u = usersById.value[userId]
  return u ? `${u.firstName} ${u.lastName}` : userId
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

    <div v-if="judgesByContest[contest.id]?.length" class="row" style="align-items: flex-start; margin-top: 14px; gap: 32px">
      <div>
        <h3 style="margin: 0 0 4px">Clicker Judges (TEx)</h3>
        <p v-if="!judgesByRole(judgesByContest[contest.id], 'clicker').length" class="muted">None assigned.</p>
        <ul v-else style="margin: 0; padding-left: 18px">
          <li v-for="a in judgesByRole(judgesByContest[contest.id], 'clicker')" :key="a.id">
            #{{ a.slot }} — {{ judgeName(a.userId) }}
          </li>
        </ul>
      </div>
      <div>
        <h3 style="margin: 0 0 4px">Evaluation Judges (PEv)</h3>
        <p v-if="!judgesByRole(judgesByContest[contest.id], 'evaluator').length" class="muted">None assigned.</p>
        <ul v-else style="margin: 0; padding-left: 18px">
          <li v-for="a in judgesByRole(judgesByContest[contest.id], 'evaluator')" :key="a.id">
            #{{ a.slot }} — {{ judgeName(a.userId) }}
          </li>
        </ul>
      </div>
    </div>
  </div>

  <p v-if="!store.loading && !store.contests.length" class="muted">
    No contests yet. Create one above, or ask a head judge to invite you.
  </p>
</template>
