<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import { theme, toggleTheme } from './composables/theme'
import { useAuthStore } from './stores/auth'
import Icon from './components/Icon.vue'

const auth = useAuthStore()
const router = useRouter()

async function handleLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}
</script>

<template>
  <nav class="app-nav">
    <RouterLink class="brand" :to="{ name: 'contests' }">yoyo-judge</RouterLink>
    <div class="user">
      <button
        class="chrome-btn"
        :title="theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
        :aria-label="theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'"
        @click="toggleTheme"
      >
        <Icon :name="theme === 'dark' ? 'sun' : 'moon'" />
      </button>
      <template v-if="auth.user">
        <span>{{ auth.user.firstName }} {{ auth.user.lastName }}</span>
        <button @click="handleLogout">Log out</button>
      </template>
    </div>
  </nav>
  <RouterView />
</template>
