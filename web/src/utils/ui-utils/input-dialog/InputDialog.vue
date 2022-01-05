<template>
  <div class="input-dialog__input-wrapper">
    <textarea
      v-if="multipleLine"
      v-model="text"
      v-focus
      class="input-dialog__input"
      :placeholder="placeholder"
      :disabled="!!loading"
    ></textarea>
    <input
      v-else
      v-model="text"
      v-focus
      class="input-dialog__input"
      :placeholder="placeholder"
      type="text"
      :disabled="!!loading"
    />
    <div v-if="validationError" class="input-dialog__validation">
      {{ validationError }}
    </div>
  </div>
</template>
<script setup>
import { val } from '@/utils'
import { ref, unref, watch } from 'vue'

/**
 * @typedef Validator
 * @property {string} trigger 'ok' or 'change'(default)
 * @property {Function} validate validate function
 * @property {RegExp} pattern pattern
 * @property {string} message message to display when pattern violated
 */

const props = defineProps({
  loading: {
    type: String,
    required: true,
  },
  opts: {
    type: Object,
    required: true,
  },
})

const emit = defineEmits(['loading'])

const text = ref(props.opts.text || '')
const placeholder = ref(props.opts.placeholder || '')
const multipleLine = ref(val(props.opts.multipleLine, false))
const validationError = ref('')

let validator = unref(props.opts.validator)

let _t

const doValidateCallback = (validate) => {
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
      (e) => {
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
      validationResult(v.message || 'Invalid input')
      return false
    }
  }
  return true
}

const beforeConfirm = async () => {
  return (await doValidate()) ? text.value : Promise.reject()
}

const validationResult = (message, token) => {
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
