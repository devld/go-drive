<template>
  <div class="site-config">
    <div class="site-config-form">
      <simple-form v-model="siteConfig" :form="siteConfigForm" />
      <simple-button :loading="siteConfigSaving" @click="saveSiteConfig">{{
        $t('p.admin.site.save')
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

const siteConfigSaving = ref(false)
const siteConfig = ref({})
const siteConfigForm = computed(() => [
  { field: 'app.name', label: t('p.admin.site.app_name'), type: 'text' },
])

const loadConfig = async () => {
  try {
    const opts = await getOptions(...siteConfigForm.value.map((f) => f.field))
    Object.assign(siteConfig.value, opts)
  } catch (e) {
    alert(e.message)
  }
}

const saveSiteConfig = async () => {
  siteConfigSaving.value = true
  try {
    await setOptions(siteConfig.value)
  } catch (e) {
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
