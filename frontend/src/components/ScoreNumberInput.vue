<script setup lang="ts">
import { ref } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: number
    min?: number
    max?: number
    step?: number
    disabled?: boolean
    width?: string
  }>(),
  { step: 1, width: '80px' },
)

const emit = defineEmits<{ 'update:modelValue': [value: number] }>()
const inputRef = ref<HTMLInputElement | null>(null)

function clamp(n: number): number {
  if (Number.isNaN(n)) return props.min ?? 0
  const lo = props.min ?? Number.NEGATIVE_INFINITY
  const hi = props.max ?? Number.POSITIVE_INFINITY
  return Math.max(lo, Math.min(hi, n))
}

function set(value: number) {
  emit('update:modelValue', clamp(value))
}

function bump(dir: 1 | -1) {
  set(props.modelValue + dir * props.step)
  // Keep focus on the input after clicking a stepper so keyboard nav
  // continues to work without a second click.
  inputRef.value?.focus()
}

// Arrow up/down move focus to the neighboring input rather than nudging
// the numeric value (that's what the on-screen ▲▼ buttons are for).
// Enter is a nice bonus - moves to the next field.
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowUp' || e.key === 'ArrowDown' || e.key === 'Enter') {
    e.preventDefault()
    focusSibling(e.key === 'ArrowUp' ? -1 : 1)
  }
}

function focusSibling(dir: 1 | -1) {
  if (!inputRef.value) return
  const all = Array.from(document.querySelectorAll<HTMLInputElement>('input.score-input'))
  const idx = all.indexOf(inputRef.value)
  if (idx < 0) return
  let next = idx + dir
  // Skip disabled siblings so navigation lands on a usable input.
  while (next >= 0 && next < all.length && all[next].disabled) next += dir
  const target = all[next]
  if (target) {
    target.focus()
    target.select()
  }
}

// Select the whole value on focus so typing replaces (rather than appends
// to) the existing number - fixes the "click 0, type 5, get 05" trap.
function onFocus(e: FocusEvent) {
  ;(e.target as HTMLInputElement).select()
}

function onInput(e: Event) {
  const raw = (e.target as HTMLInputElement).value
  // Let the user clear the field mid-edit - we'll snap it to a valid
  // number on blur.
  if (raw === '' || raw === '-') return
  const n = Number(raw)
  if (Number.isFinite(n)) set(n)
}

function onBlur() {
  if (inputRef.value && inputRef.value.value === '') set(0)
}
</script>

<template>
  <div class="score-input-wrap" :class="{ 'is-disabled': disabled }">
    <input
      ref="inputRef"
      class="score-input"
      type="text"
      inputmode="numeric"
      autocomplete="off"
      :value="modelValue"
      :disabled="disabled"
      :style="{ width }"
      @keydown="onKeydown"
      @focus="onFocus"
      @input="onInput"
      @blur="onBlur"
    />
    <div class="score-steppers">
      <button
        type="button"
        class="score-step"
        tabindex="-1"
        aria-label="Increase"
        :disabled="disabled"
        @mousedown.prevent
        @click="bump(1)"
      >
        ▲
      </button>
      <button
        type="button"
        class="score-step"
        tabindex="-1"
        aria-label="Decrease"
        :disabled="disabled"
        @mousedown.prevent
        @click="bump(-1)"
      >
        ▼
      </button>
    </div>
  </div>
</template>
