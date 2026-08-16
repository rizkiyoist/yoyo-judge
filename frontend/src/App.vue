<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { theme, toggleTheme } from './composables/theme'
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
    <div class="user">
      <button :title="theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'" @click="toggleTheme">
        {{ theme === 'dark' ? '☀️' : '🌙' }}
      </button>
      <template v-if="auth.user">
        <span>{{ auth.user.firstName }} {{ auth.user.lastName }}</span>
        <button @click="handleLogout">Log out</button>
      </template>
    </div>
  </nav>
  <RouterView />
</template>
