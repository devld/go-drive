<template>
  <div class="form-item" :class="{ error: !!error, required: item.required }">
    <span v-if="item.label" class="label">
      <span>{{ item.label }}</span>
      <span v-if="item.required" class="form-item-required">*</span>
      <a
        v-if="item.description"
        class="form-item-help"
        href="javascript:;"
        :title="s(item.description)"
        @click="toggleHelpShowing"
      >
        <Icon svg="#icon-help" />
      </a>
    </span>
    <span
      v-if="item.description && (!item.label || helpShowing)"
      class="description"
    >
      {{ item.description }}
    </span>
    <div class="value-wrapper">
      <div v-if="slots.value" class="value full-width">
        <slot name="value" />
      </div>
      <textarea
        v-if="item.type === 'textarea'"
        class="value full-width"
        :name="item.field"
        :value="modelValue"
        :placeholder="s(item.placeholder)"
        :required="item.required"
        :disabled="item.disabled"
        rows="4"
        @input="textInput"
      />
      <input
        v-if="item.type === 'text'"
        class="value full-width"
        type="text"
        :name="item.field"
        :value="modelValue"
        :placeholder="s(item.placeholder)"
        :required="item.required"
        :disabled="item.disabled"
        @input="textInput"
      />
      <input
        v-if="item.type === 'password'"
        class="value full-width"
        type="password"
        :name="item.field"
        :value="modelValue"
        :placeholder="s(item.placeholder)"
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
        class="value full-width"
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
          :title="s(o.title)"
          :disabled="o.disabled"
        >
          {{ o.name }}
        </option>
      </select>
      <FormItemForm
        v-if="item.type === 'form'"
        ref="typeFormEl"
        class="value full-width"
        :item="item"
        :model-value="modelValue"
        @update:model-value="formInput"
      />
    </div>
    <span v-if="error" class="form-item-error">{{ error }}</span>
  </div>
</template>
<script setup lang="ts">
import { isT } from '@/i18n';
import { FormItem } from '@/types'
import { ref, useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import FormItemForm from './FormItemForm.vue'

const props = defineProps({
  modelValue: {
    type: String,
  },
  item: {
    type: Object as PropType<FormItem>,
    required: true,
  },
})

const slots = useSlots()

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const error = ref<I18nText | null>(null)

const { t } = useI18n()

const helpShowing = ref(false)
const toggleHelpShowing = () => {
  helpShowing.value = !helpShowing.value
}

const typeFormEl = ref<InstanceType<typeof FormItemForm>>()

const validate = async () => {
  if (props.item.type === 'form' && typeFormEl.value) {
    try {
      await typeFormEl.value.validate()
    } catch (e: any) {
      error.value = e.message
      throw e
    }
  }
  if (props.item.required && !props.modelValue) {
    error.value = t('form.required_msg', { f: props.item.label })
    throw new Error(error.value)
  }
  if (typeof props.item.validate === 'function') {
    const err = await props.item.validate(props.modelValue)
    if (typeof err === 'string' || isT(err)) {
      error.value = err
      throw new Error(error.value.toString())
    }
  }
  return props.modelValue
}

const clearError = () => {
  error.value = null
}

defineExpose({ clearError, validate })

const formInput = (e: string) => {
  emit('update:modelValue', e)
  clearError()
}

const textInput = (e: Event) => {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
  clearError()
}

const checkboxInput = (e: Event) => {
  emit('update:modelValue', (e.target as HTMLInputElement).checked ? '1' : '')
  clearError()
}

const selectInput = (e: Event) => {
  emit('update:modelValue', (e.target as HTMLSelectElement).value)
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

.form-item {
  .value-wrapper {
    align-self: stretch;
  }

  .full-width {
    width: 100%;
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
