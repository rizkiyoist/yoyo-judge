import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'
import type { User } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)

  async function init() {
    if (initialized.value) return
    user.value = await api.currentUser()
    initialized.value = true
  }

  async function login(email: string): Promise<boolean> {
    const found = await api.login(email)
    user.value = found
    return found !== null
  }

  async function logout() {
    await api.logout()
    user.value = null
  }

  return { user, initialized, init, login, logout }
})
