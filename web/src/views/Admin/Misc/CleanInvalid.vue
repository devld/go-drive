<template>
  <div class="section">
    <h1 class="section-title">{{ $t('p.admin.misc.clean_invalid') }}</h1>
    <simple-button :loading="cleaning" @click="cleanPermissionsAndMounts">
      {{ $t('p.admin.misc.clean') }}
    </simple-button>
  </div>
</template>
<script setup>
import { cleanPermissionsAndMounts as cleanPermissionsAndMountsApi } from '@/api/admin'
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const cleaning = ref(false)

const cleanPermissionsAndMounts = async () => {
  cleaning.value = true
  try {
    const n = await cleanPermissionsAndMountsApi()
    alert(t('p.admin.misc.invalid_path_cleaned', { n: n }))
  } catch (e) {
    alert(e.message)
  } finally {
    cleaning.value = false
  }
}
</script>
