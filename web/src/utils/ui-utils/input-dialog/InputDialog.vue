<template>
  <div class="input-dialog__input-wrapper">
    <textarea
      v-if="multipleLine"
      v-model="text"
      v-focus
      class="input-dialog__input"
      :placeholder="s(placeholder)"
      :disabled="!!loading"
    ></textarea>
    <input
      v-else
      v-model="text"
      v-focus
      :type="opts.type || 'text'"
      class="input-dialog__input"
      :placeholder="s(placeholder)"
      :disabled="!!loading"
    />
    <div v-if="validationError" class="input-dialog__validation">
      {{ validationError }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { s } from '@/i18n'
import { val } from '@/utils'
import { ref, unref, watch } from 'vue'
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
  background-color: var(--form-value-bg-color);
  border: var(--form-value-border);
  color: var(--primary-text-color);
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
