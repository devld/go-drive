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
import { computed, ref, unref, watch } from 'vue'
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
// Dialog focus is retried after the open transition starts, and an earlier
// focus() is dropped. Re-apply until the user edits so only the focus that
// sticks wins, and a later automatic refocus does not clear their edit.
let selectionLocked = false
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

const applySelection = (el: HTMLInputElement | HTMLTextAreaElement) => {
  const range = props.opts.select
  if (!range) return
  const length = el.value.length
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
  el.setSelectionRange(start, end)
}

const lockSelection = () => {
  selectionLocked = true
}

const onFieldFocus = () => {
  if (selectionLocked || !props.opts.select) return
  const el = fieldEl.value
  if (!el) return
  requestAnimationFrame(() => {
    if (selectionLocked || fieldEl.value !== el) return
    if (document.activeElement !== el) return
    applySelection(el)
  })
}

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
