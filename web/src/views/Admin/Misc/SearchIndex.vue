<template>
  <div class="section">
    <h1 class="section-title">{{ $t('p.admin.misc.search_index') }}</h1>
    <div class="search-index-submit">
      <simple-form-item
        v-model="indexPath"
        :item="indexPathForm"
        class="search-index-path"
      />
      <simple-button :loading="indexSubmitting" @click="submitIndex">
        {{ $t('p.admin.misc.search_submit_index') }}
      </simple-button>
    </div>

    <div class="search-index-tasks">
      <table class="simple-table">
        <thead>
          <tr>
            <th>{{ $t('p.admin.misc.search_th_path') }}</th>
            <th>{{ $t('p.admin.misc.search_th_status') }}</th>
            <th>{{ $t('p.admin.misc.search_th_created_at') }}</th>
            <th>{{ $t('p.admin.misc.search_th_updated_at') }}</th>
            <th>{{ $t('p.admin.misc.search_th_ops') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="task.id">
            <td>{{ task.name }}</td>
            <td class="center">{{ taskStatus(task) }}</td>
            <td class="center">{{ formatTime(task.createdAt) }}</td>
            <td class="center">{{ formatTime(task.updatedAt) }}</td>
            <td>
              <simple-button
                type="danger"
                :loading="tasks.opLoading"
                :disabled="isTaskFinished(task)"
                @click="stopTask(task)"
                >{{ $t('p.admin.misc.search_index_stop') }}</simple-button
              >
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
<script setup>
import { deleteTask, getTasks } from '@/api'
import { searchIndex } from '@/api/admin'
import { useInterval } from '@/utils/hooks/timer'
import { alert } from '@/utils/ui-utils'
import { formatTime } from '@/utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const indexPathForm = computed(() => ({
  type: 'text',
  placeholder: t('p.admin.misc.search_path_tips'),
}))
const indexPath = ref('')
const indexSubmitting = ref(false)

const tasks = ref([])

const isTaskFinished = (task) =>
  ['done', 'error', 'canceled'].includes(task.status)

const taskStatus = (task) =>
  `${t(`app.task_status_${task.status}`)}${
    isTaskFinished(task)
      ? ''
      : ` (${task.progress.loaded}/${task.progress.total || '-'})`
  }`

let tasksLoading = false
const loadTasks = async () => {
  if (tasksLoading) return
  tasksLoading = true
  try {
    const ts = await getTasks('search')
    ts.forEach((task) => {
      task.opLoading = false
    })
    ts.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
    tasks.value = ts
  } catch (e) {
    alert(e.message)
  } finally {
    tasksLoading = false
  }
}

const stopTask = async (task) => {
  task.opLoading = true
  try {
    await deleteTask(task.id)
    loadTasks()
  } catch (e) {
    alert(e.message)
  } finally {
    task.opLoading = false
  }
}

const submitIndex = async () => {
  indexSubmitting.value = true
  try {
    await searchIndex(indexPath.value)
    loadTasks()
  } catch (e) {
    alert(e.message)
  } finally {
    indexSubmitting.value = false
  }
}

useInterval(
  () => {
    loadTasks()
  },
  5000,
  true
)
</script>
<style lang="scss">
.search-index-submit {
  display: flex;
  margin-bottom: 16px;

  .search-index-path > input.value {
    width: 100%;
  }
}

.search-index-path {
  overflow: hidden;
  flex: 1;
  margin-bottom: 0;
  padding-right: 16px;
}
</style>
