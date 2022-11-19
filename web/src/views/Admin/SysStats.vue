<template>
  <div class="sys-stats">
    <div class="header">
      <SimpleButton :loading="statLoading" @click="loadStats">
        {{ $t('p.admin.misc.refresh_in', { n: refreshCountDown }) }}
      </SimpleButton>
    </div>
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
</template>
<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { loadStats as loadStatsApi } from '@/api/admin'
import { alert } from '@/utils/ui-utils'
import { ServiceStatsItem } from '@/types'

const stats = ref<ServiceStatsItem[]>([])
const refreshCountDown = ref(0)
const statLoading = ref(false)

const loadStats = async () => {
  stopStatTimer()
  if (statLoading.value) return
  statLoading.value = true
  try {
    stats.value = await loadStatsApi()
  } catch (e: any) {
    await alert(e.message)
  } finally {
    statLoading.value = false
    startStatTimer()
  }
}

let timer: number

const startStatTimer = async () => {
  if (statLoading.value) return
  refreshCountDown.value = 10
  timer = setInterval(statRefreshTimer, 1000) as unknown as number
}

const stopStatTimer = () => {
  clearInterval(timer)
}

const statRefreshTimer = () => {
  refreshCountDown.value--
  if (refreshCountDown.value <= 0) {
    loadStats()
  }
}

loadStats()

onBeforeUnmount(() => {
  stopStatTimer()
})
</script>
<style lang="scss">
.sys-stats {
  padding: 16px;

  .header {
    margin: 0 0 16px;
  }
}

.statistics {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
}

.stat-item {
  margin: 0 2em 2em 0;
}
</style>
