<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const TOKEN_KEY = 'yoyo-judge-token'

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const token = params.get('token')
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
    // Reset so the router guard's auth.init() re-fetches with the new token.
    auth.invalidate()
  }
  router.replace('/')
})
</script>

<template>
  <div class="card" style="max-width: 420px; margin: 40px auto">
    <p class="muted">Finishing sign-in…</p>
  </div>
</template>
