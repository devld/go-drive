<template>
  <div class="input-dialog__input-wrapper">
    <textarea
      class="input-dialog__input"
      v-if="multipleLine"
      v-model="text"
      :disabled="!!loading"
      v-focus
    ></textarea>
    <input
      class="input-dialog__input"
      v-else
      type="text"
      v-model="text"
      :disabled="!!loading"
      v-focus
    />
    <div v-if="validationError" class="input-dialog__validation">
      {{ validationError }}
    </div>
  </div>
</template>
<script>
import { val } from '@/utils'

/**
 * @typedef Validator
 * @property {string} trigger 'ok' or 'change'(default)
 * @property {Function} validate validate function
 * @property {RegExp} pattern pattern
 * @property {string} message message to display when pattern violated
 */

export default {
  name: 'InputDialogInner',
  props: {
    loading: {
      type: String,
      required: true,
    },
    opts: {
      type: Object,
      required: true,
    },
  },
  data() {
    return {
      text: '',
      placeholder: '',
      multipleLine: false,

      validationError: '',
    }
  },
  watch: {
    text() {
      this.clearValidationResult()
      if (this._validator && this._validator.trigger !== 'confirm') {
        this.doValidate()
      }
    },
  },
  created() {
    this.text = this.opts.text || ''
    this.placeholder = this.opts.placeholder || ''
    this.multipleLine = val(this.opts.multipleLine, false)

    this._validator = this.opts.validator
  },
  methods: {
    async beforeConfirm() {
      // eslint-disable-next-line prefer-promise-reject-errors
      return (await this.doValidate()) ? this.text : Promise.reject()
    },
    doValidate() {
      const v = this._validator
      if (!v) return true
      if (typeof v.validate === 'function') {
        return this.doValidateCallback(v.validate)
      }
      if (v.pattern instanceof RegExp) {
        if (!v.pattern.test(this.text)) {
          this.validationResult(v.message || 'Invalid input')
          return false
        }
      }
      return true
    },
    doValidateCallback(validate) {
      const r = validate(this.text)
      if (r && typeof r.then === 'function') {
        this.$emit('loading', true)
        if (!this._t) this._t = 0
        const token = ++this._t
        return r.then(
          () => {
            this.validationResult(null, token)
            this.$emit('loading')
            return true
          },
          e => {
            this.validationResult(e, token)
            this.$emit('loading')
            return false
          }
        )
      } else {
        this.validationResult(r)
        return !!r
      }
    },
    validationResult(message, token) {
      if (token !== undefined && token !== this._t) return
      if (!message) {
        this.clearValidationResult()
        return
      }
      if (typeof message === 'string') this.validationError = message
      if (typeof message === 'object' && typeof message.message === 'string') {
        this.validationError = message.message
      }
    },
    clearValidationResult() {
      this.validationError = null
    },
  },
}
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
