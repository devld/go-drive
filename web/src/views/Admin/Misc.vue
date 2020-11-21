<template>
  <div class="misc-settings">
    <div class="section">
      <h1 class="section-title">
        Permission of root
        <simple-button
          @click="savePermissions"
          :loading="saving"
          :disabled="!permissionsCanSave"
        >
          Save
        </simple-button>
      </h1>
      <permissions-editor
        ref="permissionsEditor"
        :path="rootPath"
        v-model="permissions"
      />
    </div>
    <div class="section">
      <h1 class="section-title">Clean invalid permissions and mounts</h1>
      <simple-button :loading="cleaning" @click="cleanPermissionsAndMounts">
        Clean
      </simple-button>
    </div>
    <div class="section">
      <h1 class="section-title">
        Statistics
        <simple-button :loading="statLoading" @click="loadStats">
          Refresh in {{ refreshCountDown }}s
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
import { cleanPermissionsAndMounts, loadStats } from '@/api/admin'
import PermissionsEditor from './PermissionsEditor'

export default {
  name: 'MiscSettings',
  components: { PermissionsEditor },
  data () {
    return {
      permissions: [],
      rootPath: '',
      saving: false,
      permissionsCanSave: true,

      cleaning: false,

      stats: [],
      refreshCountDown: 0,
      statLoading: false
    }
  },
  watch: {
    permissions: {
      deep: true,
      handler () {
        this.permissionsCanSave = this.$refs.permissionsEditor.validate()
      }
    }
  },
  created () {
    this.loadStats()
  },
  beforeDestroy () {
    this.stopStatTimer()
  },
  methods: {
    async savePermissions () {
      this.saving = true
      try {
        await this.$refs.permissionsEditor.save()
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.saving = false
      }
    },
    async cleanPermissionsAndMounts () {
      this.cleaning = true
      try {
        const n = await cleanPermissionsAndMounts()
        this.$alert(`${n} invalid paths cleaned`)
      } catch (e) {
        this.$alert(e.message)
      } finally {
        this.cleaning = false
      }
    },
    async loadStats () {
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
    startStatTimer () {
      this.refreshCountDown = 10
      this._timer = setInterval(this.statRefreshTimer, 1000)
    },
    stopStatTimer () {
      clearInterval(this._timer)
    },
    statRefreshTimer () {
      this.refreshCountDown--
      if (this.refreshCountDown <= 0) {
        this.loadStats()
        this.stopStatTimer()
      }
    }
  }
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
      @include var(border-color, border-color);
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
}
</style>
