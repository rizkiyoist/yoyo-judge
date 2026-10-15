<script setup lang="ts">
import GoogleLogo from '../components/GoogleLogo.vue'

const basePath = (import.meta.env.VITE_BASE_PATH ?? '/yoyojudge').replace(/\/$/, '')
const apiBase = import.meta.env.VITE_API_BASE_URL ?? `${basePath}/api`
const googleLoginUrl = `${apiBase}/auth/google`
</script>

<template>
  <div class="card login-card">
    <h2>Log in</h2>

    <a :href="googleLoginUrl" class="google-btn">
      <GoogleLogo :size="18" />
      <span>Continue with Google</span>
    </a>
  </div>
</template>
