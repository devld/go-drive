<template>
  <div
    class="form-item"
    :class="{ error: !!error, required: item.required, disabled }"
  >
    <Component
      :is="valueId ? 'label' : 'span'"
      v-if="item.label"
      :for="valueId"
      class="label"
    >
      <span>{{ item.label }}</span>
      <span v-if="item.required" class="form-item-required">*</span>
      <a
        v-if="item.description"
        class="form-item-help"
        href="javascript:;"
        :title="s(item.description)"
        @click.stop="toggleHelpShowing"
      >
        <Icon svg="#icon-help" />
      </a>

      <span v-if="slots['label-suffix']" class="form-item-suffix" @click.stop>
        <slot name="label-suffix" />
      </span>
    </Component>
    <span
      v-if="item.description && (!item.label || helpShowing)"
      class="description"
    >
      {{ item.description }}
    </span>
    <div class="value-wrapper">
      <div v-if="slots.value" class="value full-width">
        <slot :id="valueId" name="value" />
      </div>
      <textarea
        v-if="item.type === 'textarea'"
        :id="valueId"
        class="value full-width"
        :name="item.field"
        :value="modelValue"
        :placeholder="s(item.placeholder)"
        :required="item.required"
        :disabled="disabled || item.disabled"
        rows="4"
        @input="textInput"
      />
      <input
        v-if="item.type === 'text'"
        :id="valueId"
        class="value full-width"
        type="text"
        :name="item.field"
        :value="modelValue"
        :placeholder="s(item.placeholder)"
        :required="item.required"
        :disabled="disabled || item.disabled"
        @input="textInput"
      />
      <input
        v-if="item.type === 'password'"
        :id="valueId"
        class="value full-width"
        type="password"
        :name="item.field"
        :value="modelValue"
        :placeholder="s(item.placeholder)"
        :required="item.required"
        :disabled="disabled || item.disabled"
        @input="textInput"
      />
      <input
        v-if="item.type === 'checkbox'"
        :id="valueId"
        class="value"
        type="checkbox"
        :name="item.field"
        :checked="!!modelValue"
        :required="item.required"
        :disabled="disabled || item.disabled"
        @input="checkboxInput"
      />
      <div v-if="item.type === 'checkboxes'" class="value full-width form-item--checkboxes">
        <label
          v-for="o in item.options"
          :key="o.value"
          class="form-item--checkbox-option"
          :class="{ disabled: disabled || item.disabled || o.disabled }"
        >
          <input
            type="checkbox"
            :name="item.field"
            :value="o.value"
            :checked="checkboxesSelectedSet.has(o.value)"
            :disabled="disabled || item.disabled || o.disabled"
            @input="checkboxesInput(o.value, ($event.target as HTMLInputElement).checked)"
          />
          <span>{{ o.name }}</span>
        </label>
      </div>
      <select
        v-if="item.type === 'select'"
        :id="valueId"
        class="value full-width"
        :name="item.field"
        :value="modelValue"
        :required="item.required"
        :disabled="disabled || item.disabled"
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
      <div v-if="item.type === 'path'" class="full-width form-item--type-path">
        <input
          :id="valueId"
          class="value full-width"
          type="text"
          :name="item.field"
          :value="modelValue"
          :placeholder="s(item.placeholder)"
          :required="item.required"
          :disabled="disabled || item.disabled"
          @input="textInput"
        />
        <button
          v-if="!disabled || item.disabled"
          class="form-item--type-path-select"
          :title="$t('form.select_path')"
          @click="selectPath"
        >
          <Icon svg="#icon-folder" />
        </button>
      </div>
      <FormItemForm
        v-if="item.type === 'form'"
        ref="typeFormEl"
        class="value full-width"
        :item="item"
        :model-value="modelValue"
        :disabled="disabled || item.disabled"
        @update:model-value="stringInput"
      />
      <CodeEditor
        v-if="item.type === 'code'"
        :model-value="modelValue"
        v-bind="item.code ?? {}"
        :disabled="disabled || item.disabled"
        @update:model-value="stringInput"
      />
    </div>
    <span v-if="error" class="form-item-error">{{ error }}</span>
  </div>
</template>
<script setup lang="ts">
import { isT } from '@/i18n'
import { FormItem } from '@/types'
import { ref, computed, useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import FormItemForm from './FormItemForm.vue'

import CodeEditor from '../CodeEditor/index.vue'
import { open } from '@/utils/ui-utils'

const props = defineProps({
  id: {
    type: String,
  },
  modelValue: {
    type: String,
  },
  item: {
    type: Object as PropType<FormItem>,
    required: true,
  },
  disabled: {
    type: Boolean,
  },
})

const slots = useSlots()

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const error = ref<I18nText | null>(null)
const valueId = computed(() => {
  if (props.item.type === 'form' || props.item.type === 'code' || props.item.type === 'checkboxes') return
  return props.id
})

const checkboxesSelectedSet = computed(() => {
  if (props.item.type !== 'checkboxes') return new Set<string>()
  const raw = (props.modelValue || '').trim()
  if (!raw) return new Set<string>()
  return new Set(raw.split(',').map((v) => v.trim()).filter(Boolean))
})

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

const selectPath = async () => {
  try {
    const selected = await open({
      type: 'dir',
      filter: props.item.pathOptions?.filter,
      title: t('form.select_path'),
    })
    emit('update:modelValue', selected.path)
    clearError()
  } catch {
    // ignore
  }
}

const stringInput = (e: string) => {
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

const checkboxesInput = (optionValue: string, checked: boolean) => {
  const set = new Set(checkboxesSelectedSet.value)
  if (checked) set.add(optionValue)
  else set.delete(optionValue)
  emit('update:modelValue', [...set].join(','))
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

.form-item--type-path {
  position: relative;

  input.value {
    padding-right: 28px !important;
  }

  &-select {
    position: absolute;
    top: 0;
    bottom: 0;
    right: 0;
    border: 0;
    outline: none;
    padding: 0 6px;
    font-size: 16px;
    cursor: pointer;
    background-color: transparent;
  }
}

.form-item.disabled .form-item--type-path {
  input.value {
    padding-right: 8px !important;
  }
}

.form-item--checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 16px;

  .form-item--checkbox-option {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    user-select: none;

    &.disabled {
      cursor: not-allowed;
      opacity: 0.6;
    }

    input {
      margin: 0;
    }
  }
}
</style>
