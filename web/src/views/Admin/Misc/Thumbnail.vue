<template>
  <div class="section">
    <h1 class="section-title">{{ $t('p.admin.misc.thumbnail_config') }}</h1>

    <div class="thumbnail-config-form">
      <simple-form
        ref="thumbnailConfigFormEl"
        v-model="thumbnailConfig"
        :form="thumbnailConfigForm"
      />
      <simple-button :loading="thumbnailSaving" @click="saveThumbnailConfig">{{
        $t('p.admin.misc.save')
      }}</simple-button>
    </div>
  </div>
</template>
<script setup>
import { getOptions, setOptions } from '@/api/admin'
import { alert } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const thumbnailConfigFormEl = ref(null)
const thumbnailSaving = ref(false)
const thumbnailConfig = ref({})
const thumbnailConfigForm = computed(() => [
  {
    field: 'thumbnail.handlersMapping',
    label: t('p.admin.misc.thumbnail_mapping'),
    description: t('p.admin.misc.thumbnail_mapping_tips'),
    placeholder: t('p.admin.misc.thumbnail_mapping_placeholder'),
    type: 'textarea',
    width: '100%',
    validate: (v) =>
      !v ||
      !v
        .split('\n')
        .filter(Boolean)
        .some((f) => !/^([A-z0-9-_](,[A-z0-9-_])*):(.+)$/.test(f)) ||
      t('p.admin.misc.thumbnail_mapping_invalid'),
  },
])

const loadConfig = async () => {
  try {
    const opts = await getOptions(
      ...thumbnailConfigForm.value.map((f) => f.field)
    )
    Object.assign(thumbnailConfig.value, opts)
  } catch (e) {
    alert(e.message)
  }
}

const saveThumbnailConfig = async () => {
  try {
    await thumbnailConfigFormEl.value.validate()
  } catch {
    return
  }
  thumbnailSaving.value = true
  try {
    await setOptions(thumbnailConfig.value)
  } catch (e) {
    alert(e.message)
  } finally {
    thumbnailSaving.value = false
  }
}

loadConfig()
</script>
