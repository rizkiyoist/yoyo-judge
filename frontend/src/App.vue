<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import { onMounted, onUnmounted, watch, ref } from 'vue'
import { theme, toggleTheme, viewMode, cycleViewMode } from './composables/theme'
import { useAuthStore } from './stores/auth'
import Icon from './components/Icon.vue'
import type { IconName } from './components/Icon.vue'

const VIEW_MODE_META: Record<string, { icon: IconName; label: string }> = {
  auto: { icon: 'wand', label: 'Auto' },
  mobile: { icon: 'smartphone', label: 'Mobile' },
  desktop: { icon: 'monitor', label: 'Desktop' },
}

const auth = useAuthStore()
const router = useRouter()
const navRef = ref<HTMLElement | null>(null)

async function handleLogout() {
  await auth.logout()
  router.push({ name: 'login' })
}

// Publish the top-nav height as --topbar-h so the workspace tab bar can
// stick flush against it even in mobile-view (which changes font-size
// and therefore nav height). Re-measure on theme + view-mode changes.
function measureTopbar() {
  const el = navRef.value
  if (el) document.documentElement.style.setProperty('--topbar-h', el.offsetHeight + 'px')
}
onMounted(() => {
  measureTopbar()
  window.addEventListener('resize', measureTopbar)
})
onUnmounted(() => window.removeEventListener('resize', measureTopbar))
watch([viewMode, theme], () => requestAnimationFrame(measureTopbar))
</script>

<template>
  <nav ref="navRef" class="app-nav">
    <RouterLink class="brand" :to="{ name: 'contests' }">yoyo-judge</RouterLink>
    <div class="user">
      <button
        class="chrome-btn"
        :title="`View: ${VIEW_MODE_META[viewMode].label} (click to change)`"
        :aria-label="`View mode: ${VIEW_MODE_META[viewMode].label}`"
        @click="cycleViewMode"
      >
        <Icon :name="VIEW_MODE_META[viewMode].icon" />
      </button>
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
