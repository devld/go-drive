<template>
  <div class="site-config">
    <div class="site-config-form">
      <SimpleForm v-model="siteConfig" :form="siteConfigForm" />
      <SimpleButton :loading="siteConfigSaving" @click="saveSiteConfig">{{
        $t('p.admin.site.save')
      }}</SimpleButton>
    </div>
  </div>
</template>
<script setup lang="ts">
import { getOptions, setOptions } from '@/api/admin'
import {
  DEFAULT_IMAGE_FILE_EXTS,
  DEFAULT_MEDIA_FILE_EXTS,
  DEFAULT_TEXT_FILE_EXTS,
} from '@/config'
import { FormItem } from '@/types'
import { alert, loading } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const siteConfigSaving = ref(false)
const siteConfig = ref<O<string>>({})
const siteConfigForm = computed<FormItem[]>(() => [
  { field: 'app.name', label: t('p.admin.site.app_name'), type: 'text' },
  {
    field: 'proxy.maxSize',
    label: t('p.admin.site.proxy_max'),
    description: t('p.admin.site.proxy_max_desc'),
    type: 'text',
  },
  {
    field: 'web.officePreviewEnabled',
    label: t('p.admin.site.office_preview_enabled'),
    description: t('p.admin.site.office_preview_enabled_desc'),
    type: 'checkbox',
  },
  {
    field: 'web.textFileExts',
    label: t('p.admin.site.text_file_exts'),
    description: t('p.admin.site.text_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_TEXT_FILE_EXTS.join(','),
  },
  {
    field: 'web.imageFileExts',
    label: t('p.admin.site.image_file_exts'),
    description: t('p.admin.site.image_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_IMAGE_FILE_EXTS.join(','),
  },
  {
    field: 'web.mediaFileExts',
    label: t('p.admin.site.media_file_exts'),
    description: t('p.admin.site.media_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_MEDIA_FILE_EXTS.join(','),
  },
])

const loadConfig = async () => {
  loading(true)
  try {
    const opts = await getOptions(...siteConfigForm.value.map((f) => f.field!))
    Object.assign(siteConfig.value, opts)

    siteConfigForm.value.forEach((item) => {
      if (!siteConfig.value[item.field!] && item.defaultValue) {
        siteConfig.value[item.field!] = item.defaultValue
      }
    })
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading()
  }
}

const saveSiteConfig = async () => {
  siteConfigSaving.value = true
  try {
    await setOptions(siteConfig.value)
  } catch (e: any) {
    alert(e.message)
  } finally {
    siteConfigSaving.value = false
  }
}

loadConfig()
</script>
<style lang="scss">
.site-config {
  padding: 16px;
}
</style>
