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
import { FormItem } from '@/types'
import { alert } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const siteConfigSaving = ref(false)
const siteConfig = ref({})
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
])

const loadConfig = async () => {
  try {
    const opts = await getOptions(...siteConfigForm.value.map((f) => f.field!))
    Object.assign(siteConfig.value, opts)
  } catch (e: any) {
    alert(e.message)
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
