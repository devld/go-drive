<template>
  <div class="maintenance-action">
    <AdminDocHeading level="subsection">
      {{ $t('p.admin.misc.clean_invalid') }}
    </AdminDocHeading>
    <SimpleButton :loading="cleaning" @click="cleanPermissionsAndMounts">
      {{ $t('p.admin.misc.clean') }}
    </SimpleButton>
  </div>
</template>
<script setup lang="ts">
import AdminDocHeading from '../AdminDocHeading.vue'
import { cleanPermissionsAndMounts as cleanPermissionsAndMountsApi } from '@/api/admin'
import { alert } from '@/utils/ui-utils'
import { ref } from 'vue'
import { useI18n } from '@go-drive/i18n'

const { t } = useI18n()

const cleaning = ref(false)

const cleanPermissionsAndMounts = async () => {
  cleaning.value = true
  try {
    const n = await cleanPermissionsAndMountsApi()
    alert(t('p.admin.misc.invalid_path_cleaned', { n: n }))
  } catch (e: any) {
    alert(e.message)
  } finally {
    cleaning.value = false
  }
}
</script>
