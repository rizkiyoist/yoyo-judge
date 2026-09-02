<script setup lang="ts">
const basePath = (import.meta.env.VITE_BASE_PATH ?? '/yoyojudge').replace(/\/$/, '')
const apiBase = import.meta.env.VITE_API_BASE_URL ?? `${basePath}/api`
const googleLoginUrl = `${apiBase}/auth/google`
</script>

<template>
  <div class="card" style="max-width: 420px; margin: 40px auto">
    <h2>Log in</h2>

    <a
      :href="googleLoginUrl"
      class="button primary"
      style="display: block; text-align: center"
    >
      Continue with Google
    </a>
  </div>
</template>
