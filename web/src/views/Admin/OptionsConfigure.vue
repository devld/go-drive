<template>
  <div class="section">
    <h1 class="section-title">
      {{ title }}
      <Icon v-if="loading" class="loading-icon" svg="#icon-loading" />
    </h1>

    <div class="config-form">
      <SimpleForm ref="configFormEl" v-model="configValue" :form="form" />
      <SimpleButton :loading="saving" @click="saveConfig">{{
        $t('p.admin.save')
      }}</SimpleButton>
    </div>
  </div>
</template>
<script setup lang="ts">
import { getOptions, setOptions } from '@/api/admin'
import { FormItem } from '@/types'
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  form: {
    type: Array as PropType<FormItem[]>,
    required: true,
  },
})

const configFormEl = ref<InstanceType<SimpleFormType> | null>(null)
const loading = ref(false)
const saving = ref(false)
const configValue = ref<O<string>>({})

const loadConfig = async () => {
  loading.value = true
  try {
    const opts = await getOptions(...props.form.map((f) => f.field!))
    Object.assign(configValue.value, opts)

    props.form.forEach((item) => {
      if (
        !configValue.value[item.field!] &&
        item.defaultValue &&
        item.fillDefaultIfEmpty
      ) {
        configValue.value[item.field!] = item.defaultValue
      }
    })
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  try {
    await configFormEl.value!.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await setOptions(configValue.value)
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

loadConfig()
</script>
