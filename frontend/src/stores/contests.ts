import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { Contest, ScoringStage } from '../types'

export const useContestStore = defineStore('contests', () => {
  const contests = ref<Contest[]>([])
  const loading = ref(false)

  async function fetchContests(userId: string) {
    loading.value = true
    try {
      contests.value = await api.listContests(userId)
    } finally {
      loading.value = false
    }
  }

  async function createContest(name: string, ownerUserId: string): Promise<Contest> {
    const contest = await api.createContest(name, ownerUserId)
    contests.value = [...contests.value, contest]
    return contest
  }

  async function addDivision(contestId: string, name: string, stages: ScoringStage[]) {
    await api.addDivision(contestId, name, stages)
    await refreshContest(contestId)
  }

  async function updateDivisionStages(contestId: string, divisionId: string, stages: ScoringStage[]) {
    await api.updateDivisionStages(contestId, divisionId, stages)
    await refreshContest(contestId)
  }

  async function refreshContest(contestId: string) {
    const updated = await api.getContest(contestId)
    if (!updated) return
    const idx = contests.value.findIndex((c) => c.id === contestId)
    if (idx === -1) contests.value = [...contests.value, updated]
    else contests.value = [...contests.value.slice(0, idx), updated, ...contests.value.slice(idx + 1)]
  }

  return { contests, loading, fetchContests, createContest, addDivision, updateDivisionStages, refreshContest }
})
