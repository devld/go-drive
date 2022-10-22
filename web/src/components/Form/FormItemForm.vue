<template>
  <div class="form-item__form">
    <div v-for="(f, i) in value" :key="i" class="form-item__form-item">
      <template v-if="formsMapByKey[f.typeKey]">
        <div class="form-item__form-item-title">
          <span class="title-text"
            >{{ i + 1 }}. {{ formsMapByKey[f.typeKey].name }}</span
          >
          <button
            class="close-button plain-button small"
            @click="removeItem(i)"
          >
            <Icon svg="#icon-close" />
          </button>
        </div>
        <Form
          ref="formsEl"
          :form="formsMapByKey[f.typeKey].form"
          :model-value="f.value"
          @update:model-value="onInput(f, $event)"
        />
      </template>
    </div>
    <div v-if="addable" class="form-item__form-add">
      <SimpleDropdown v-model="addDropdownShowing">
        <SimpleButton icon="#icon-add">{{ forms.addText }}</SimpleButton>
        <template #dropdown>
          <ul class="form-item__form-types">
            <li
              v-for="s in forms.forms"
              :key="s.key"
              class="form-item__form-type"
              @click="addForm(s.key)"
            >
              {{ s.name }}
            </li>
          </ul>
        </template>
      </SimpleDropdown>
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
})

const emit = defineEmits<{
  (e: 'update:modelValue', v: string): void
}>()

const formsEl = ref<InstanceType<typeof SimpleForm>[]>([])

const addDropdownShowing = ref(false)

const value = ref<ValueItem[]>([])

const forms = computed(() => props.item.forms || { forms: [] })
const maxItems = computed(() => forms.value.maxItems ?? 0)

const addable = computed(
  () => maxItems.value === 0 || value.value.length < maxItems.value
)

const formsMapByKey = computed(() => mapOf(forms.value.forms, (e) => e.key))

const addForm = (typeKey: string) => {
  value.value.push({
    typeKey,
    value: {},
  })
  addDropdownShowing.value = false
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
      let obj = JSON.parse(v!)
      if (!Array.isArray(obj)) {
        obj = obj ? [obj] : []
      }
      if (maxItems.value === 1) obj.splice(1)
      value.value = obj
        .filter((e: O) => e && !!e.$key && typeof e === 'object')
        .map((e: O) => {
          const { $key, ...others } = e
          return { typeKey: $key, value: others }
        })
    } catch (e) {
      console.error(e)
    }
  },
  { immediate: true }
)

const emitValue = debounce(() => {
  const v = value.value.map((e) => ({ $key: e.typeKey, ...e.value }))
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
  border: solid 1px var(--border-color);

  & > .simple-form {
    padding: 4px 10px;
  }
}

.form-item__form-item-title {
  margin-bottom: 8px;
  border-bottom: solid 1px var(--border-color);
  padding: 4px 10px;
  display: flex;
  align-items: center;

  .title-text {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    flex: 1;
  }

  .close-button {
    margin-left: auto;
  }
}

.form-item__form-types {
  margin: 0;
  padding: 0;
  max-height: 100px;
  overflow-y: auto;
}

.form-item__form-type {
  margin: 0;
  list-style-type: none;
  white-space: nowrap;
  padding: 6px 12px;
  cursor: pointer;
  font-size: 14px;

  &:hover {
    background-color: var(--hover-bg-color);
  }

  &.active {
    background-color: var(--select-bg-color);
  }
}
</style>
