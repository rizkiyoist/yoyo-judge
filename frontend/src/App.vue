<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useRouter } from 'vue-router'

const auth = useAuthStore()
const router = useRouter()

async function handleLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <nav class="app-nav">
    <RouterLink class="brand" :to="{ name: 'contests' }">🪀 yoyo-judge</RouterLink>
    <div v-if="auth.user" class="user">
      <span>{{ auth.user.firstName }} {{ auth.user.lastName }}</span>
      <button @click="handleLogout">Log out</button>
    </div>
  </nav>
  <RouterView />
</template>
