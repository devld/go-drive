<template>
  <div class="section">
    <h1 class="section-title">{{ $t('p.admin.misc.clean_cache') }}</h1>
    <SimpleFormItem
      v-model="cacheSelectedDrive"
      class="cache-clean-form-item"
      :item="drivesForm"
    />
    <SimpleButton
      :loading="cacheCleaning"
      :disabled="!cacheSelectedDrive"
      @click="cleanDriveCache"
    >
      {{ $t('p.admin.misc.clean') }}
    </SimpleButton>
  </div>
</template>
<script setup lang="ts">
import { getDrives, cleanDriveCache as cleanDriveCacheApi } from '@/api/admin'
import { Drive, FormItem } from '@/types'
import { alert } from '@/utils/ui-utils'
import { computed, ref } from 'vue'

const cacheSelectedDrive = ref('')
const drives = ref<Drive[]>([])
const cacheCleaning = ref(false)

const drivesForm = computed<FormItem>(() => ({
  type: 'select',
  options: [
    { name: '', value: '' },
    ...drives.value.map((d) => ({ name: d.name, value: d.name })),
  ],
}))

const loadDrives = async () => {
  try {
    drives.value = await getDrives()
  } catch (e: any) {
    alert(e.message)
  }
}

const cleanDriveCache = async () => {
  cacheCleaning.value = true
  try {
    await cleanDriveCacheApi(cacheSelectedDrive.value)
  } catch (e: any) {
    await alert(e.message)
  } finally {
    cacheCleaning.value = false
  }
}

loadDrives()
</script>
<style lang="scss">
.cache-clean-form-item {
  display: inline-block;
  margin-right: 1em;
}
</style>
