<template>
  <div class="form-item" :class="{ error: !!error, required: item.required }">
    <span v-if="item.label" class="label">
      <span>{{ item.label }}</span>
      <span v-if="item.required" class="form-item-required">*</span>
      <a
        v-if="item.description"
        class="form-item-help"
        href="javascript:;"
        :title="item.description"
        @click="toggleHelpShowing"
      >
        <i-icon svg="#icon-help" />
      </a>
    </span>
    <span v-if="helpShowing" class="description">
      {{ item.description }}
    </span>
    <div v-if="slots.value" class="value">
      <slot name="value" />
    </div>
    <textarea
      v-if="item.type === 'textarea'"
      class="value"
      :name="item.field"
      :value="modelValue"
      :placeholder="item.placeholder"
      :required="item.required"
      :disabled="item.disabled"
      rows="4"
      @input="textInput"
    />
    <input
      v-if="item.type === 'text'"
      class="value"
      type="text"
      :name="item.field"
      :value="modelValue"
      :placeholder="item.placeholder"
      :required="item.required"
      :disabled="item.disabled"
      @input="textInput"
    />
    <input
      v-if="item.type === 'password'"
      class="value"
      type="password"
      :name="item.field"
      :value="modelValue"
      :placeholder="item.placeholder"
      :required="item.required"
      :disabled="item.disabled"
      @input="textInput"
    />
    <input
      v-if="item.type === 'checkbox'"
      class="value"
      type="checkbox"
      :name="item.field"
      :checked="!!modelValue"
      :required="item.required"
      :disabled="item.disabled"
      @input="checkboxInput"
    />
    <select
      v-if="item.type === 'select'"
      class="value"
      :name="item.field"
      :value="modelValue"
      :required="item.required"
      :disabled="item.disabled"
      @input="selectInput"
    >
      <option
        v-for="o in item.options"
        :key="o.value"
        :value="o.value"
        :title="o.title"
        :disabled="o.disabled"
      >
        {{ o.name }}
      </option>
    </select>
    <span v-if="error" class="form-item-error">{{ error }}</span>
  </div>
</template>
<script setup>
import { ref, useSlots } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  modelValue: {
    type: [String, Number],
  },
  item: {
    type: Object,
    required: true,
  },
})

const slots = useSlots()

const emit = defineEmits(['update:modelValue'])

const error = ref(null)

const { t } = useI18n()

const helpShowing = ref(false)
const toggleHelpShowing = () => {
  helpShowing.value = !helpShowing.value
}

const validate = async () => {
  if (props.item.required && !props.modelValue) {
    error.value = t('form.required_msg', { f: props.item.label })
    throw new Error(error.value)
  }
  if (typeof props.item.validate === 'function') {
    const err = await props.item.validate(props.modelValue)
    if (typeof err === 'string') {
      error.value = err
      throw new Error(error.value)
    }
  }
  return props.modelValue
}

const clearError = () => {
  error.value = null
}

defineExpose({ clearError, validate })

const textInput = (e) => {
  emit('update:modelValue', e.target.value)
  clearError()
}

const checkboxInput = (e) => {
  emit('update:modelValue', e.target.checked ? '1' : '')
  clearError()
}

const selectInput = (e) => {
  emit('update:modelValue', e.target.value)
  clearError()
}
</script>
<style lang="scss">
.form-item.error {
  position: relative;
  padding-bottom: 24px;

  .value {
    border: solid 1px red;
  }
}

.form-item-required {
  color: red;
}

.form-item-error {
  position: absolute;
  bottom: 0;
  right: 16px;
  color: red;
}

.form-item-help {
  margin-left: 0.5em;
  text-decoration: none;
  color: inherit;
  cursor: help;
}
</style>
