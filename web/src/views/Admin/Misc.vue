<template>
  <div class="misc-settings">
    <div class="section">
      <h1 class="section-title">
        {{ $t('p.admin.misc.permission_of_root') }}
        <simple-button
          :loading="saving"
          :disabled="!permissionsCanSave"
          @click="savePermissions"
        >
          {{ $t('p.admin.misc.save') }}
        </simple-button>
      </h1>
      <permissions-editor
        ref="permissionsEditorEl"
        v-model="permissions"
        :path="rootPath"
      />
    </div>
    <div class="section">
      <h1 class="section-title">{{ $t('p.admin.misc.clean_invalid') }}</h1>
      <simple-button :loading="cleaning" @click="cleanPermissionsAndMounts">
        {{ $t('p.admin.misc.clean') }}
      </simple-button>
    </div>
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
    <div class="section">
      <h1 class="section-title">
        {{ $t('p.admin.misc.statistics') }}
        <simple-button :loading="statLoading" @click="loadStats">
          {{ $t('p.admin.misc.refresh_in', { n: refreshCountDown }) }}
        </simple-button>
      </h1>
      <div class="statistics">
        <table v-for="(s, i) in stats" :key="i" class="stat-item simple-table">
          <thead>
            <tr>
              <th colspan="2">{{ s.name }}</th>
            </tr>
          </thead>
          <tr v-for="(value, key) in s.data" :key="key">
            <td>{{ key }}</td>
            <td>{{ value }}</td>
          </tr>
        </table>
      </div>
    </div>
  </div>
</template>
<script setup>
import {
  cleanDriveCache as cleanDriveCacheApi,
  cleanPermissionsAndMounts as cleanPermissionsAndMountsApi,
  getDrives,
  loadStats as loadStatsApi,
} from '@/api/admin'
import PermissionsEditor from './PermissionsEditor.vue'
import { alert } from '@/utils/ui-utils'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const permissions = ref([])
const rootPath = ref('')
const saving = ref(false)
const permissionsCanSave = ref(true)

const cleaning = ref(false)

const drives = ref([])
const cacheSelectedDrive = ref(null)
const cacheCleaning = ref(false)

const stats = ref([])
const refreshCountDown = ref(0)
const statLoading = ref(false)

const permissionsEditorEl = ref(null)

let timer

const drivesForm = computed(() => ({
  type: 'select',
  options: [
    { name: '', value: '' },
    ...drives.value.map((d) => ({ name: d.name, value: d.name })),
  ],
}))

const savePermissions = async () => {
  saving.value = true
  try {
    await permissionsEditorEl.value.save()
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

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

const loadStats = async () => {
  statLoading.value = true
  try {
    stats.value = await loadStatsApi()
  } catch (e) {
    await alert(e.message)
  } finally {
    statLoading.value = false
    startStatTimer()
  }
}

const startStatTimer = async () => {
  refreshCountDown.value = 10
  timer = setInterval(statRefreshTimer, 1000)
}

const stopStatTimer = () => {
  clearInterval(timer)
}

const statRefreshTimer = () => {
  refreshCountDown.value--
  if (refreshCountDown.value <= 0) {
    loadStats()
    stopStatTimer()
  }
}

watch(
  () => permissions.value,
  () => {
    permissionsCanSave.value = permissionsEditorEl.value.validate()
  },
  { deep: true }
)

loadDrives()
loadStats()

onBeforeUnmount(() => {
  stopStatTimer()
})
</script>
<style lang="scss">
.misc-settings {
  padding: 16px;

  .section {
    padding-top: 1em;
    margin-bottom: 2em;

    &:not(:first-child) {
      border-top: solid 1px;
      border-color: var(--border-color);
    }
  }

  .section-title {
    margin: 0 0 16px;
    font-size: 20px;
    font-weight: normal;
  }

  .statistics {
    display: flex;
    flex-wrap: wrap;
  }

  .stat-item {
    margin: 0 2em 2em 0;
  }

  .cache-clean-form-item {
    display: inline-block;
    margin-right: 1em;
  }
}
</style>
