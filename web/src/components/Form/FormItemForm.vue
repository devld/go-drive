<template>
  <div class="form-item__form">
    <div v-for="(f, i) in value" :key="i" class="form-item__form-item">
      <template v-if="formsMapByKey[f.typeKey]">
        <div class="form-item__form-item-title">
          <span class="title-text">
            {{ i + 1 }}<template v-if="formsMapByKey[f.typeKey].name"
              >. {{ formsMapByKey[f.typeKey].name }}</template
            >
          </span>
          <SimpleButton
            class="close-button"
            variant="plain"
            small
            icon="close"
            :aria-label="t('dialog.base.close')"
            @click="removeItem(i)"
          />
        </div>
        <Form
          ref="formsEl"
          :form="formsMapByKey[f.typeKey].form"
          :model-value="f.value"
          :disabled="disabled"
          @update:model-value="onInput(f, $event)"
        />
      </template>
    </div>
    <div v-if="!disabled && addable && forms.forms.length">
      <SimpleDropdown
        v-if="forms.forms.length > 1"
        position="bottom-right"
        :aria-label="forms.addText?.toString()"
        :items="forms.forms"
        menu-max-height="100px"
        trigger-class="simple-button small outline"
        @select="addForm"
      >
        <Icon name="add" aria-hidden="true" />
        <span v-if="forms.addText">{{ forms.addText }}</span>
      </SimpleDropdown>
      <SimpleButton
        v-else
        small
        variant="outline"
        icon="add"
        :aria-label="forms.addText?.toString()"
        @click="addForm(forms.forms[0].key)"
      >
        <template v-if="forms.addText">{{ forms.addText }}</template>
      </SimpleButton>
    </div>
  </div>
</template>
<script setup lang="ts">
import { FormItem } from '@/types'
import { debounce, mapOf } from '@/utils'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SimpleForm from '.'
import Form from './index.vue'

const { t } = useI18n()

interface ValueItem {
  typeKey: string
  value: O
}

const props = defineProps({
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

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const formsEl = ref<InstanceType<typeof SimpleForm>[]>([])

const value = ref<ValueItem[]>([])

const forms = computed(() => props.item.forms || { forms: [] })
const maxItems = computed(() => forms.value.maxItems ?? 0)
const keyField = computed(() => forms.value.keyField ?? '$key')
const valueField = computed(() => forms.value.valueField)

const addable = computed(
  () => maxItems.value === 0 || value.value.length < maxItems.value
)

const formsMapByKey = computed(() => mapOf(forms.value.forms, (e) => e.key))

const addForm = (typeKey: string) => {
  value.value.push({
    typeKey,
    value: {},
  })
  emitValue()
}

const removeItem = (i: number) => {
  value.value.splice(i, 1)
  emitValue()
}

const onInput = (v: ValueItem, data: O) => {
  v.value = data
  emitValue()
}

let lastValue = ''

watch(
  () => props.modelValue,
  (v) => {
    if (v === lastValue) return
    try {
      let obj = v ? JSON.parse(v) : undefined
      if (!Array.isArray(obj)) {
        obj = obj ? [obj] : []
      }
      if (maxItems.value === 1) obj.splice(1)
      value.value = obj
        .filter((e: O) => e && typeof e === 'object' && !!e[keyField.value])
        .map((e: O) => ({
          typeKey: e[keyField.value],
          value: valueField.value
            ? e[valueField.value]
            : { ...e, [keyField.value]: undefined },
        }))
    } catch (e) {
      console.error(e)
    }
  },
  { immediate: true }
)

const emitValue = debounce(() => {
  const v = value.value.map((e) => {
    return {
      [keyField.value]: e.typeKey,
      ...(valueField.value ? { [valueField.value]: e.value } : e.value),
    }
  })
  lastValue = JSON.stringify(maxItems.value === 1 ? v[0] : v)
  emit('update:modelValue', lastValue)
}, 100)

const validate = async () => {
  if (props.item.required && value.value.length === 0) {
    throw new Error(t('form.required_msg', { f: props.item.label }))
  }
  return await Promise.all(formsEl.value.map((e) => e.validate()))
}

defineExpose({ validate })
</script>
<style lang="scss">
.form-item__form-item {
  margin-bottom: 16px;
  border: solid 1px var(--color-border);
  border-radius: var(--radius-control);
  background-color: var(--color-field-bg);
  -webkit-backdrop-filter: var(--backdrop-filter-field);
  backdrop-filter: var(--backdrop-filter-field);
  overflow: hidden;

  & > .simple-form {
    padding: 4px 10px;
  }
}

.form-item__form-item-title {
  margin-bottom: 8px;
  border-bottom: solid 1px var(--color-border);
  padding: 4px 10px;
  display: flex;
  align-items: center;

  .title-text {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    flex: 1;
  }

  .close-button.simple-button {
    margin-left: auto;
  }
}

</style>
