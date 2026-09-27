<template>
  <div class="input-dialog__input-wrapper">
    <textarea
      v-if="multipleLine"
      ref="fieldEl"
      v-model="text"
      v-focus
      class="input-dialog__input"
      :placeholder="s(placeholder)"
      :aria-label="inputLabel"
      :aria-invalid="validationError ? 'true' : undefined"
      :aria-describedby="validationError ? validationId : undefined"
      :disabled="!!loading"
      @focus="onFieldFocus"
      @pointerdown="lockSelection"
      @input="lockSelection"
    ></textarea>
    <input
      v-else
      ref="fieldEl"
      v-model="text"
      v-focus
      :type="opts.type || 'text'"
      class="input-dialog__input"
      :placeholder="s(placeholder)"
      :aria-label="inputLabel"
      :aria-invalid="validationError ? 'true' : undefined"
      :aria-describedby="validationError ? validationId : undefined"
      :disabled="!!loading"
      @focus="onFieldFocus"
      @pointerdown="lockSelection"
      @input="lockSelection"
    />
    <div
      v-if="validationError"
      :id="validationId"
      class="input-dialog__validation"
      role="alert"
    >
      {{ validationError }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { s } from '@go-drive/i18n'
import { val } from '@/utils'
import { computed, onBeforeUnmount, onMounted, ref, unref, watch } from 'vue'
import { InputDialogOptions, InputDialogValidateFunc } from '.'

const props = defineProps({
  loading: {
    type: String,
    required: true,
  },
  opts: {
    type: Object as PropType<InputDialogOptions>,
    required: true,
  },
})

const emit = defineEmits<{ (e: 'loading', v?: boolean): void }>()

const text = ref(props.opts.text || '')
const placeholder = ref(props.opts.placeholder || '')
const fieldEl = ref<HTMLInputElement | HTMLTextAreaElement | null>(null)
// Dialog focus is retried after the open transition, and an earlier focus()
// is dropped. Selecting the whole value during that focus is collapsed to a
// caret, so the range is applied again once focus has settled.
let selectionLocked = false
let selectionTimers: number[] = []
let selectionAttempts = 0
const validationId = `input-dialog-validation-${Math.round(Math.random() * 1000000)}`
const inputLabel = computed(() =>
  s(props.opts.title || placeholder.value)
)
const multipleLine = ref(val(props.opts.multipleLine, false))
const validationError = ref<string | null>('')

const validator = unref(props.opts.validator)

let _t: number

const doValidateCallback = (validate: InputDialogValidateFunc) => {
  const r = validate(text.value)
  if (r && typeof r.then === 'function') {
    emit('loading', true)
    if (!_t) _t = 0
    const token = ++_t
    return r.then(
      () => {
        validationResult(null, token)
        emit('loading')
        return true
      },
      (e: string | Error) => {
        validationResult(e, token)
        emit('loading')
        return false
      }
    )
  } else {
    validationResult(r)
    return !!r
  }
}

const doValidate = () => {
  const v = validator
  if (!v) return true
  if (typeof v.validate === 'function') {
    return doValidateCallback(v.validate)
  }
  if (v.pattern instanceof RegExp) {
    if (!v.pattern.test(text.value)) {
      validationResult(s(v.message) || 'Invalid input')
      return false
    }
  }
  return true
}

const beforeConfirm = async () => {
  return (await doValidate()) ? text.value : Promise.reject()
}

const validationResult = (message: string | Error | null, token?: number) => {
  if (token !== undefined && token !== _t) return
  if (!message) {
    clearValidationResult()
    return
  }
  if (typeof message === 'string') validationError.value = message
  if (typeof message === 'object' && typeof message.message === 'string') {
    validationError.value = message.message
  }
}
const clearValidationResult = () => {
  validationError.value = null
}

const selectionBounds = (length: number) => {
  const range = props.opts.select
  if (!range) return
  const normalize = (index: number) => {
    if (!Number.isFinite(index)) return 0
    return Math.min(length, Math.max(0, Math.trunc(index)))
  }
  let start = normalize(range.start)
  let end = normalize(range.end)
  if (start > end) {
    const swap = start
    start = end
    end = swap
  }
  return { start, end }
}

const applySelection = (el: HTMLInputElement | HTMLTextAreaElement) => {
  const bounds = selectionBounds(el.value.length)
  if (!bounds) return
  el.setSelectionRange(bounds.start, bounds.end)
}

const clearSelectionTimers = () => {
  selectionTimers.forEach((id) => window.clearTimeout(id))
  selectionTimers = []
}

const lockSelection = () => {
  selectionLocked = true
  clearSelectionTimers()
}

const applySelectionIfFocused = () => {
  if (selectionLocked || !props.opts.select) return
  const el = fieldEl.value
  if (!el || document.activeElement !== el) return
  applySelection(el)
}

// Selecting the whole value during the dialog's focus sequence is collapsed
// to a caret. Repeat briefly after focus settles, and stop if the user edits.
const selectionMatches = (el: HTMLInputElement | HTMLTextAreaElement) => {
  const bounds = selectionBounds(el.value.length)
  if (!bounds) return true
  return el.selectionStart === bounds.start && el.selectionEnd === bounds.end
}

const keepSelection = () => {
  if (selectionLocked || !props.opts.select || selectionAttempts > 8) return
  const el = fieldEl.value
  if (el && document.activeElement === el && selectionMatches(el)) return
  selectionAttempts += 1
  applySelectionIfFocused()
  selectionTimers.push(window.setTimeout(keepSelection, 50))
}

const onFieldFocus = () => {
  if (selectionLocked || !props.opts.select || selectionAttempts > 0) return
  keepSelection()
}

onMounted(() => {
  if (!props.opts.select) return
  keepSelection()
})

onBeforeUnmount(clearSelectionTimers)

watch(
  () => text.value,
  () => {
    clearValidationResult()
    if (validator && validator.trigger !== 'confirm') {
      doValidate()
    }
  }
)

defineExpose({ beforeConfirm })
</script>
<style lang="scss">
.input-dialog__input-wrapper {
  text-align: center;
  padding: 16px;
}

.input-dialog__input {
  background-color: var(--color-field-bg);
  border: solid 1px var(--color-field-border);
  color: var(--color-text);
  font-size: 16px;
  outline: none;
  padding: 6px;
}

.input-dialog__validation {
  color: red;
  text-align: right;
  padding-top: 16px;
}
</style>
