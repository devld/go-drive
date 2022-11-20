<template>
  <form
    class="simple-form"
    :autocomplete="noAutoComplete ? 'off' : 'on'"
    @submit="onSubmit"
  >
    <template v-for="item in form" :key="item.field">
      <div
        v-if="item.type === 'md'"
        v-markdown="item.description"
        class="form-item form-markdown markdown-body"
      ></div>

      <FormItem
        v-else
        :ref="addFieldsRef"
        v-model="data[item.field!]"
        :item="item"
        :class="item.class"
        :disabled="disabled"
        :style="{
          width:
            typeof item.width === 'number' ? `${item.width}px` : item.width,
        }"
        @update:model-value="emitInput"
      >
        <template v-if="item.slot" #value>
          <slot :name="item.slot" />
        </template>
      </FormItem>
    </template>
  </form>
</template>
<script lang="ts">
export default { name: 'FormView' }
</script>
<script setup lang="ts">
import { FormItem as FormItemType } from '@/types'
import { ComponentPublicInstance, onBeforeUpdate, ref, watch } from 'vue'
import FormItem from './FormItem.vue'

const props = defineProps({
  form: {
    type: Array as PropType<FormItemType[]>,
    required: true,
  },
  modelValue: {
    type: Object as PropType<O>,
  },
  noAutoComplete: {
    type: Boolean,
  },
  disabled: {
    type: Boolean,
  },
})

const data = ref<O>({})
let fields: InstanceType<typeof FormItem>[] = []

const emit = defineEmits<{ (e: 'update:modelValue', v: O): void }>()

const addFieldsRef = (el: Element | ComponentPublicInstance | null) => {
  if (el) fields.push(el as InstanceType<typeof FormItem>)
}
onBeforeUpdate(() => {
  fields = []
})

watch(
  () => props.modelValue,
  (val) => {
    if (val === data.value) return
    data.value = val || {}
  },
  { immediate: true }
)

const validate = async () => {
  await Promise.all(fields.map((f) => f.validate()))
}

const clearError = () => {
  fields.forEach((f) => {
    f.clearError()
  })
}

defineExpose({ validate, clearError })

const onSubmit = (e: Event) => e.preventDefault()

const emitInput = () => emit('update:modelValue', data.value)

const fillDefaultValue = () => {
  if (props.modelValue) return
  const dat = {} as O
  for (const f of props.form) {
    if (f.field) {
      dat[f.field] = f.defaultValue || null
    }
  }
  data.value = dat
  emitInput()
}

fillDefaultValue()
</script>
<style lang="scss">
.simple-form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;

  .form-item {
    width: 232px;
  }

  .form-markdown {
    width: 100%;
  }
}
</style>
