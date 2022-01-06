<template>
  <form
    class="simple-form"
    :autocomplete="noAutoComplete ? 'off' : 'on'"
    @submit="onSubmit"
  >
    <form-item
      v-for="item in form"
      :key="item.field"
      :ref="setFieldsRef"
      v-model="data[item.field]"
      :item="item"
      @update:model-value="emitInput"
    />
  </form>
</template>
<script>
export default { name: 'FormView' }
</script>
<script setup>
import { onBeforeUpdate, ref, watch } from 'vue'
import FormItem from './FormItem.vue'

const props = defineProps({
  form: {
    type: Array,
    required: true,
  },
  modelValue: {
    type: Object,
  },
  noAutoComplete: {
    type: Boolean,
  },
})

const data = ref({})
const fields = ref([])

const emit = defineEmits(['update:modelValue'])

const setFieldsRef = (el) => fields.value.push(el)
onBeforeUpdate(() => {
  fields.value = []
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
  await Promise.all(fields.value.map((f) => f.validate()))
}

const clearError = () => {
  fields.value.forEach((f) => {
    f.clearError()
  })
}

defineExpose({ validate, clearError })

const onSubmit = (e) => e.preventDefault()

const emitInput = () => emit('update:modelValue', data.value)

const fillDefaultValue = () => {
  if (props.modelValue) return
  const dat = {}
  for (const f of props.form) {
    dat[f.field] = f.defaultValue || null
  }
  console.log(JSON.stringify(dat))
  data.value = dat
  emitInput()
}

fillDefaultValue()
</script>
<style lang="scss">
.simple-form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;

  .form-item {
    width: 232px;
  }
}
</style>
