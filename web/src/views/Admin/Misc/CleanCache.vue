<template>
  <div class="section">
    <h1 class="section-title">{{ $t('p.admin.misc.clean_cache') }}</h1>
    <simple-form-item
      v-model="cacheSelectedDrive"
      class="cache-clean-form-item"
      :item="drivesForm"
    />
    <simple-button
      :loading="cacheCleaning"
      :disabled="!cacheSelectedDrive"
      @click="cleanDriveCache"
    >
      {{ $t('p.admin.misc.clean') }}
    </simple-button>
  </div>
</template>
<script setup>
import { getDrives, cleanDriveCache as cleanDriveCacheApi } from '@/api/admin'
import { alert } from '@/utils/ui-utils'
import { computed, ref } from 'vue'

const cacheSelectedDrive = ref(null)
const drives = ref([])
const cacheCleaning = ref(false)

const drivesForm = computed(() => ({
  type: 'select',
  options: [
    { name: '', value: '' },
    ...drives.value.map((d) => ({ name: d.name, value: d.name })),
  ],
}))

const loadDrives = async () => {
  try {
    drives.value = await getDrives()
  } catch (e) {
    alert(e.message)
  }
}

const cleanDriveCache = async () => {
  cacheCleaning.value = true
  try {
    await cleanDriveCacheApi(cacheSelectedDrive.value)
  } catch (e) {
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
