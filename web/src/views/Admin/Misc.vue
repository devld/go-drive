<template>
  <div class="misc-settings">
    <div class="section">
      <h1 class="section-title">
        {{ $t('p.admin.misc.permission_of_root') }}
        <simple-button
          @click="savePermissions"
          :loading="saving"
          :disabled="!permissionsCanSave"
        >
          {{ $t('p.admin.misc.save') }}
        </simple-button>
      </h1>
      <permissions-editor
        ref="permissionsEditor"
        :path="rootPath"
        v-model="permissions"
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
        class="cache-clean-form-item"
        :item="drivesForm"
        v-model="cacheSelectedDrive"
      />
      <simple-button
        :loading="cacheCleaning"
        @click="cleanDriveCache"
        :disabled="!cacheSelectedDrive"
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
        <table class="stat-item simple-table" v-for="(s, i) in stats" :key="i">
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
<script>
import {
  cleanDriveCache,
  cleanPermissionsAndMounts,
  getDrives,
  loadStats,
} from '@/api/admin'
import PermissionsEditor from './PermissionsEditor'

export default {
  name: 'MiscSettings',
  components: { PermissionsEditor },
  data() {
    return {
      permissions: [],
      rootPath: '',
      saving: false,
      permissionsCanSave: true,

      cleaning: false,

      drives: [],
      cacheSelectedDrive: null,
      cacheCleaning: false,

      stats: [],
      refreshCountDown: 0,
      statLoading: false,
    }
  },
  computed: {
    drivesForm() {
      return {
        type: 'select',
        options: [
          { name: '', value: '' },
          ...this.drives.map(d => ({ name: d.name, value: d.name })),
        ],
      }
    },
  },
  watch: {
    permissions: {
      deep: true,
      handler() {
        this.permissionsCanSave = this.$refs.permissionsEditor.validate()
      },
    },
  },
  created() {
    this.loadDrives()
    this.loadStats()
  },
  beforeDestroy() {
    this.stopStatTimer()
  },
  methods: {
    async savePermissions() {
      this.saving = true
      try {
        await this.$refs.permissionsEditor.save()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
    async cleanPermissionsAndMounts() {
      this.cleaning = true
      try {
        const n = await cleanPermissionsAndMounts()
        this.$alert(this.$t('p.admin.misc.invalid_path_cleaned', { n: n }))
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.cleaning = false
      }
    },
    async loadDrives() {
      try {
        this.drives = await getDrives()
      } catch (e) {
        this.$alert(e.message)
      }
    },
    async cleanDriveCache() {
      this.cacheCleaning = true
      try {
        await cleanDriveCache(this.cacheSelectedDrive)
      } catch (e) {
        await this.$alert(e.message)
      } finally {
        this.cacheCleaning = false
      }
    },
    async loadStats() {
      this.statLoading = true
      try {
        this.stats = await loadStats()
      } catch (e) {
        await this.$alert(e.message)
      } finally {
        this.statLoading = false
        this.startStatTimer()
      }
    },
    startStatTimer() {
      this.refreshCountDown = 10
      this._timer = setInterval(this.statRefreshTimer, 1000)
    },
    stopStatTimer() {
      clearInterval(this._timer)
    },
    statRefreshTimer() {
      this.refreshCountDown--
      if (this.refreshCountDown <= 0) {
        this.loadStats()
        this.stopStatTimer()
      }
    },
  },
}
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
