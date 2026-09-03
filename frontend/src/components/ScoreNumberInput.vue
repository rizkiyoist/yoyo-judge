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
    // Identifies this input's position within its table so arrow-key nav
    // can move to the same column in the row above/below (Up/Down/Enter)
    // or the neighboring column in the same row (Left/Right). group scopes
    // navigation to one table - inputs in the clicker table never jump
    // into the eval table, say - and is required for arrow nav to work at
    // all (omit it on a one-off input with no siblings to navigate to).
    group?: string
    row?: number
    col?: number
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

// Moves focus to the input at (row + dRow, col + dCol) within the same
// group, if one exists - the mechanics behind Up/Down/Left/Right/Enter
// below.
function focusAt(dRow: number, dCol: number) {
  if (props.row === undefined || props.col === undefined) return
  const targetRow = props.row + dRow
  const targetCol = props.col + dCol
  const all = Array.from(document.querySelectorAll<HTMLInputElement>('input.score-input'))
  const target = all.find(
    (el) =>
      el.dataset.group === (props.group ?? '') &&
      Number(el.dataset.row) === targetRow &&
      Number(el.dataset.col) === targetCol &&
      !el.disabled,
  )
  if (target) {
    target.focus()
    target.select()
  }
}

// Arrow keys navigate the score grid like a spreadsheet rather than
// nudging the numeric value or moving the text cursor (that's what the
// on-screen ▲▼ buttons and typing are for): Up/Down move to the same
// column in the row above/below, Left/Right to the neighboring column in
// the same row, and Enter moves down a row for fast top-to-bottom entry
// through one column.
function onKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'ArrowUp':
      e.preventDefault()
      focusAt(-1, 0)
      break
    case 'ArrowDown':
    case 'Enter':
      e.preventDefault()
      focusAt(1, 0)
      break
    case 'ArrowLeft':
      e.preventDefault()
      focusAt(0, -1)
      break
    case 'ArrowRight':
      e.preventDefault()
      focusAt(0, 1)
      break
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
      :data-group="group"
      :data-row="row"
      :data-col="col"
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
