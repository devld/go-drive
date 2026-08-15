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
        <Icon name="help" />
      </a>

      <span v-if="slots['label-suffix']" @click.stop>
        <slot name="label-suffix" />
      </span>
    </Component>
    <span
      v-if="item.description && (!item.label || helpShowing)"
      class="description"
    >
      <template v-for="(part, i) in descriptionParts" :key="i">
        <a
          v-if="part.href"
          :href="part.href"
          target="_blank"
          rel="noreferrer noopener"
          @click.stop
          >{{ part.text }}</a
        >
        <template v-else>{{ part.text }}</template>
      </template>
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
        :aria-label="s(item.ariaLabel)"
        :autocomplete="item.autocomplete"
        :required="item.required"
        :disabled="disabled || item.disabled"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? errorId : undefined"
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
        :aria-label="s(item.ariaLabel)"
        :autocomplete="item.autocomplete"
        :required="item.required"
        :disabled="disabled || item.disabled"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? errorId : undefined"
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
        :aria-label="s(item.ariaLabel)"
        :autocomplete="item.autocomplete"
        :required="item.required"
        :disabled="disabled || item.disabled"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? errorId : undefined"
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
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? errorId : undefined"
        @input="checkboxInput"
      />
      <div
        v-if="item.type === 'checkboxes'"
        class="value full-width form-item--checkboxes"
      >
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
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? errorId : undefined"
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
          :aria-invalid="error ? 'true' : undefined"
          :aria-describedby="error ? errorId : undefined"
          @input="textInput"
        />
        <SimpleButton
          v-if="!(disabled || item.disabled)"
          class="form-item--type-path-select"
          variant="plain"
          small
          icon="folder"
          :title="$t('form.select_path')"
          :aria-label="$t('form.select_path')"
          @click="selectPath"
        />
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
    <span
      v-if="error"
      :id="errorId"
      class="form-item-error"
      role="alert"
    >{{ error }}</span>
  </div>
</template>
<script setup lang="ts">
import { isT, s } from '@/i18n'
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

interface DescriptionPart {
  text: string
  href?: string
}

const countCharacter = (value: string, character: string) =>
  value.split(character).length - 1

const splitUrlTrailingText = (value: string): [string, string] => {
  let urlEnd = value.length
  while (urlEnd > 0 && /[.,;:!?，。；：！？、]/u.test(value[urlEnd - 1])) {
    urlEnd--
  }

  const pairs = [
    ['(', ')'],
    ['[', ']'],
    ['{', '}'],
    ['（', '）'],
    ['【', '】'],
    ['《', '》'],
  ] as const
  for (const [open, close] of pairs) {
    while (
      value[urlEnd - 1] === close &&
      countCharacter(value.slice(0, urlEnd), close) >
        countCharacter(value.slice(0, urlEnd), open)
    ) {
      urlEnd--
    }
  }

  return [value.slice(0, urlEnd), value.slice(urlEnd)]
}

const parseDescription = (description: string): DescriptionPart[] => {
  const parts: DescriptionPart[] = []
  const urlPattern = /https?:\/\/[^\s<>"']+/giu
  let textStart = 0

  for (const match of description.matchAll(urlPattern)) {
    const matchStart = match.index ?? 0
    if (matchStart > textStart) {
      parts.push({ text: description.slice(textStart, matchStart) })
    }

    const [url, trailingText] = splitUrlTrailingText(match[0])
    parts.push({ text: url, href: url })
    if (trailingText) parts.push({ text: trailingText })
    textStart = matchStart + match[0].length
  }

  if (textStart < description.length) {
    parts.push({ text: description.slice(textStart) })
  }
  return parts
}

const slots = useSlots()

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const error = ref<I18nText | null>(null)
const descriptionParts = computed(() =>
  parseDescription(s(props.item.description) ?? '')
)
const valueId = computed(() => {
  if (props.item.type === 'form' || props.item.type === 'code' || props.item.type === 'checkboxes') return
  return props.id
})
const errorId = computed(() =>
  props.id ? `${props.id}-error` : undefined
)

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
    border-color: var(--color-danger);
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
  color: var(--color-danger);
}

.form-item-error {
  position: absolute;
  bottom: 0;
  right: 16px;
  color: var(--color-danger);
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

  &-select.simple-button.plain {
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
    color: var(--color-text-muted);
    border-radius: 0 var(--radius-control) var(--radius-control) 0;

    &:hover {
      background-color: var(--color-bg-hover);
      color: var(--color-text);
    }
  }
}

.form-item .code-editor {
  box-sizing: border-box;
  background-color: var(--color-field-bg);
  -webkit-backdrop-filter: var(--backdrop-filter-field);
  backdrop-filter: var(--backdrop-filter-field);
  border: solid 1px var(--color-field-border);
  border-radius: var(--radius-control);
  overflow: hidden;
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
