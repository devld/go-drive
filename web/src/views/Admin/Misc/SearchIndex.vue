<template>
  <div class="section">
    <h1 class="section-title">{{ $t('p.admin.misc.search_index') }}</h1>

    <template v-if="searchEnabled">
      <div class="search-index-submit">
        <SimpleForm
          ref="indexFormEl"
          v-model="indexOptions"
          :form="indexOptionsForm"
        >
          <template #submit>
            <SimpleButton :loading="indexSubmitting" @click="submitIndex">
              {{ $t('p.admin.misc.search_submit_index') }}
            </SimpleButton>
          </template>
        </SimpleForm>
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
              <td class="line">
                <span
                  class="search-index-op"
                  :class="`search-index-op-${task.group.split('/')[1]}`"
                  >{{ searchIndexType[task.group] }}</span
                >
                <span class="search-index-op-path">{{ task.name }}</span>
              </td>
              <td class="center line">{{ taskStatus(task) }}</td>
              <td class="center line">{{ formatTime(task.createdAt) }}</td>
              <td class="center line">{{ formatTime(task.updatedAt) }}</td>
              <td class="center line">
                <SimpleButton
                  v-if="!isTaskFinished(task)"
                  type="danger"
                  :loading="task.opLoading"
                  @click="stopTask(task)"
                  >{{ $t('p.admin.misc.search_index_stop') }}</SimpleButton
                >
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <div v-else class="search-index-disabled-tip">
      {{ $t('p.admin.misc.search_disabled') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import { deleteTask, getTasks } from '@/api'
import { getOptions, searchIndex, setOptions } from '@/api/admin'
import { useInterval } from '@/utils/hooks/timer'
import { alert } from '@/utils/ui-utils'
import { formatTime } from '@/utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/store'
import { FormItem, Task as Task_ } from '@/types'

interface Task extends Task_<void> {
  opLoading?: boolean
}

const store = useAppStore()

const searchConfig = computed(() => store.config?.search)
const searchEnabled = computed(() => searchConfig.value?.enabled)

const filterOptionKey = 'search.filter'

const { t } = useI18n()

const indexFormEl = ref<InstanceType<SimpleFormType> | null>(null)
const indexOptionsForm = computed<FormItem[]>(() => [
  {
    field: 'filters',
    type: 'textarea',
    label: t('p.admin.misc.search_form_filter'),
    description: t('p.admin.misc.search_form_filter_desc'),
    placeholder: t('p.admin.misc.search_form_filter_placeholder'),
    width: '100%',
    validate: (v: string) =>
      !v ||
      !v
        .split('\n')
        .filter(Boolean)
        .some((f) => f[0] !== '+' && f[0] !== '-') ||
      t('p.admin.misc.search_form_filter_invalid'),
  },
  {
    field: 'path',
    type: 'text',
    label: t('p.admin.misc.search_form_path'),
    description: t('p.admin.misc.search_form_path_desc'),
  },
  { slot: 'submit', class: 'flex-align-self-end' },
])
const indexOptions = ref({ path: '', filters: '' })
const indexSubmitting = ref(false)

const tasks = ref<Task[]>([])

const searchIndexType: O<any> = {
  'search/index': t('p.admin.misc.search_op_index'),
  'search/delete': t('p.admin.misc.search_op_delete'),
}

const isTaskFinished = (task: Task) =>
  ['done', 'error', 'canceled'].includes(task.status)

const taskStatus = (task: Task) =>
  `${t(`app.task_status_${task.status}`)}` +
  (task.progress
    ? ` (${task.progress.loaded}/${task.progress.total || '-'})`
    : '')

let tasksLoading = false
const loadTasks = async () => {
  if (!searchEnabled.value) return
  if (tasksLoading) return
  tasksLoading = true
  try {
    const ts = (await getTasks('search')) as Task[]
    ts.forEach((task) => {
      task.opLoading = false
    })
    ts.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
    tasks.value = ts
  } catch (e: any) {
    alert(e.message)
  } finally {
    tasksLoading = false
  }
}

const stopTask = async (task: Task) => {
  task.opLoading = true
  try {
    await deleteTask(task.id)
    loadTasks()
  } catch (e: any) {
    alert(e.message)
  } finally {
    task.opLoading = false
  }
}

let oldFilter: string
const loadIndexFilters = async () => {
  try {
    oldFilter = (await getOptions(filterOptionKey))[filterOptionKey]
    indexOptions.value.filters = oldFilter
  } catch (e: any) {
    alert(e.message)
  }
}

const saveIndexFilters = async () => {
  try {
    await indexFormEl.value!.validate()
  } catch {
    return false
  }
  if (oldFilter === indexOptions.value.filters) return
  await setOptions({ [filterOptionKey]: indexOptions.value.filters })
  oldFilter = indexOptions.value.filters
}

const submitIndex = async () => {
  indexSubmitting.value = true
  try {
    if ((await saveIndexFilters()) === false) return
    await searchIndex(indexOptions.value.path)
    indexOptions.value.path = ''
    loadTasks()
  } catch (e: any) {
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

loadIndexFilters()
</script>
<style lang="scss">
.search-index-submit {
  margin-bottom: 16px;
}

.search-index-tasks {
  position: relative;
  font-size: 14px;
  overflow: auto hidden;

  .simple-table {
    width: 100%;
  }

  th:last-child,
  td:last-child {
    position: sticky;
    right: 0;
  }
}

.search-index-op,
.search-index-op-path {
  font-size: 12px;
}

.search-index-op {
  margin-right: 0.5em;

  &-index {
    color: #1890ff;
  }

  &-delete {
    color: #f5222d;
  }
}

.search-index-op-path {
  border: solid 1px var(--secondary-text-color);
  padding: 2px 4px;
  border-radius: 4px;
}
</style>
