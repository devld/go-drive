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
import { computed, onMounted, ref, unref, watch } from 'vue'
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

// The dialog focuses this field on a later frame. Setting a selection that
// covers the whole value in that same turn is collapsed to a caret, so wait
// until the frame after that focus.
const applySelection = () => {
  const el = fieldEl.value
  const range = props.opts.select
  if (!el || !range || document.activeElement !== el) return
  const length = el.value.length
  const clamp = (index: number) =>
    Math.min(
      length,
      Math.max(0, Number.isFinite(index) ? Math.trunc(index) : 0)
    )
  let start = clamp(range.start)
  let end = clamp(range.end)
  if (start > end) [start, end] = [end, start]
  el.setSelectionRange(start, end)
}

onMounted(() => {
  if (!props.opts.select) return
  requestAnimationFrame(() => requestAnimationFrame(applySelection))
})

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
