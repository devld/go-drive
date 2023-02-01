<template>
  <div class="options-configure">
    <details
      v-for="(form, i) in forms"
      :key="i"
      class="options-group"
      :open="form.defaultOpen"
      @toggle="onDetailsToggle(i)"
    >
      <summary>
        <h3 class="options-group-title">
          {{ form.title }}
          <Icon v-if="loading" class="loading-icon" svg="#icon-loading" />
        </h3>
      </summary>

      <div v-if="detailsInited[i]" class="options-form">
        <SimpleForm
          ref="configFormElsSet"
          v-model="configValues[i]"
          :form="form.form"
        />
      </div>
    </details>

    <SimpleButton :loading="saving" :disabled="loading" @click="saveConfig">{{
      $t('p.admin.save')
    }}</SimpleButton>
  </div>
</template>
<script setup lang="ts">
import { getOptions, setOptions } from '@/api/admin'
import { FormItem } from '@/types'
import { alert } from '@/utils/ui-utils'
import { onUpdated, ref, watch } from 'vue'

export interface OptionsForm {
  title?: I18nText
  form: FormItem[]
  defaultOpen?: boolean
}

const props = defineProps({
  forms: {
    type: Object as PropType<OptionsForm[]>,
    required: true,
  },
})

const configFormEls: InstanceType<SimpleFormType>[] = []
const configFormElsSet = (el: InstanceType<SimpleFormType>) =>
  configFormEls.push(el)

onUpdated(() => configFormEls.splice(0))

const detailsInited = ref({} as Record<number, boolean>)

const loading = ref(false)
const saving = ref(false)
const configValues = ref<O<string>[]>([])

const loadConfig = async () => {
  const value = {} as O<string>
  props.forms.forEach(() => {
    configValues.value.push(value)
  })

  loading.value = true
  try {
    const opts = await getOptions(
      ...props.forms.flatMap((f) => f.form.map((ff) => ff.field!))
    )

    props.forms.forEach((form, i) => {
      const value = configValues.value[i]

      form.form.forEach((item) => {
        value[item.field!] = opts[item.field!]
      })

      form.form.forEach((item) => {
        if (
          !value[item.field!] &&
          item.defaultValue &&
          item.fillDefaultIfEmpty
        ) {
          value[item.field!] = item.defaultValue
        }
      })
    })
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading.value = false
  }
}

const saveConfig = async () => {
  try {
    await Promise.all(configFormEls.map((f) => f.validate()))
  } catch {
    return
  }
  saving.value = true
  try {
    await setOptions(
      configValues.value.reduce((p, c) => Object.assign(p, c), {})
    )
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const onDetailsToggle = (i: number) => {
  detailsInited.value[i] = true
}

const init = () => {
  loadConfig()

  detailsInited.value = {}
  props.forms.forEach((f, i) => {
    if (f.defaultOpen) detailsInited.value[i] = true
  })
}

watch(() => props.forms, init)
init()
</script>

<style lang="scss">
.options-configure {
  .options-group {
    &:not(:last-child) {
      margin-bottom: 16px;
    }
  }

  .options-group-title {
    display: inline-block;
    margin: 0 0 16px;
    font-size: 18px;
    font-weight: normal;
    cursor: pointer;
  }
}
</style>
