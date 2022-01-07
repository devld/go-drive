<template>
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
</template>
<script setup>
import { onBeforeUnmount, ref } from 'vue'
import { loadStats as loadStatsApi } from '@/api/admin'
import { alert } from '@/utils/ui-utils'

const stats = ref([])
const refreshCountDown = ref(0)
const statLoading = ref(false)

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

let timer

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

loadStats()

onBeforeUnmount(() => {
  stopStatTimer()
})
</script>
<style lang="scss">
.statistics {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
}

.stat-item {
  margin: 0 2em 2em 0;
}
</style>
