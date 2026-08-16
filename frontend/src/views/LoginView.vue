<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'
import type { User } from '../types'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const email = ref('')
const error = ref('')
const seededUsers = ref<User[]>([])

onMounted(async () => {
  seededUsers.value = await api.searchUsers('example.com')
})

async function submit(chosenEmail?: string) {
  error.value = ''
  const target = chosenEmail ?? email.value
  const ok = await auth.login(target)
  if (!ok) {
    error.value = `No user found for "${target}".`
    return
  }
  const redirect = (route.query.redirect as string) || '/'
  router.push(redirect)
}
</script>

<template>
  <div class="card" style="max-width: 420px; margin: 40px auto">
    <h2>Log in</h2>
    <p class="muted">
      There's no real backend yet — this is a mocked login. Enter a seeded user's email, or pick one below.
    </p>

    <form class="field" @submit.prevent="submit()">
      <label for="email">Email</label>
      <input id="email" v-model="email" type="email" placeholder="alice@example.com" autocomplete="email" />
      <button class="primary" type="submit" style="margin-top: 8px">Log in</button>
    </form>
    <p v-if="error" class="error">{{ error }}</p>

    <h3 style="margin-top: 24px">Seeded demo users</h3>
    <div class="row" style="flex-direction: column; align-items: stretch">
      <button v-for="u in seededUsers" :key="u.id" @click="submit(u.email)">
        {{ u.firstName }} {{ u.lastName }} <span class="muted">— {{ u.email }}</span>
      </button>
    </div>
  </div>
</template>
