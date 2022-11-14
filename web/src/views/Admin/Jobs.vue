<template>
  <div class="jobs-manager" :class="{ editing: !!jobEdit }">
    <div class="jobs-list">
      <div class="actions">
        <SimpleButton
          icon="#icon-add"
          :title="$t('p.admin.jobs.add_job')"
          @click="addJob"
        />
      </div>
      <div class="simple-table-wrapper">
        <table class="simple-table">
          <thead>
            <tr>
              <th>{{ $t('p.admin.jobs.desc') }}</th>
              <th>{{ $t('p.admin.jobs.schedule') }}</th>
              <th>{{ $t('p.admin.jobs.next_run') }}</th>
              <th>{{ $t('p.admin.jobs.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="j in jobsList"
              :key="j.id"
              :class="{ 'job-disabled': !j.enabled }"
            >
              <td class="center">{{ j.description }}</td>
              <td class="center line">{{ j.schedule }}</td>
              <td class="center">{{ j.nextRun && formatTime(j.nextRun) }}</td>
              <td class="center line">
                <SimpleButton
                  :title="$t('p.admin.jobs.view_log')"
                  small
                  icon="#icon-list"
                  @click="showJobExecutions(j)"
                />
                <SimpleButton
                  :title="$t('p.admin.jobs.edit')"
                  small
                  icon="#icon-edit"
                  @click="editJob(j)"
                />
                <SimpleButton
                  v-if="!jobEdit"
                  :title="$t('p.admin.jobs.delete')"
                  type="danger"
                  small
                  icon="#icon-delete"
                  @click="deleteJob(j)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    <div v-if="jobEdit" class="job-edit">
      <div class="small-title">
        {{ edit ? $t('p.admin.jobs.edit_job') : $t('p.admin.jobs.add_job') }}
      </div>
      <SimpleForm ref="jobFormEl" v-model="jobEdit" :form="jobForm" />
      <SimpleForm
        ref="jobParamsFormEl"
        :key="jobEdit.job"
        v-model="jobParams"
        :form="jobParamsForm"
      />
      <div class="save-button">
        <SimpleButton small :loading="saving" @click="saveJob">
          {{ $t('p.admin.jobs.save') }}
        </SimpleButton>
        <SimpleButton small type="info" @click="cancelEdit">
          {{ $t('p.admin.jobs.cancel') }}
        </SimpleButton>
      </div>
    </div>
    <div v-else-if="jobExecutionsShowing" class="job-executions">
      <div class="small-title">
        <span class="text">{{
          $t('p.admin.jobs.job_executions', {
            n: jobExecutionsShowing.description,
          })
        }}</span>
        <SimpleButton :loading="executionsClearing" @click="clearExecutions"
          >清理</SimpleButton
        >
      </div>
      <div class="simple-table-wrapper">
        <table class="simple-table">
          <thead>
            <tr>
              <th>{{ $t('p.admin.jobs.status') }}</th>
              <th>{{ $t('p.admin.jobs.started_at') }}</th>
              <th>{{ $t('p.admin.jobs.completed_at') }}</th>
              <th>{{ $t('p.admin.jobs.execution_duration') }}</th>
              <th>{{ $t('p.admin.jobs.logs') }}</th>
              <th>{{ $t('p.admin.jobs.error_msg') }}</th>
              <th>{{ $t('p.admin.jobs.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="e in jobExecutions" :key="e.id">
              <td class="center" :class="`status-${e.status}`">
                {{ STATUS_TEXTS[e.status] }}
              </td>
              <td class="center">{{ formatTime(e.startedAt) }}</td>
              <td class="center">
                {{ (e.completedAt && formatTime(e.completedAt)) || '' }}
              </td>
              <td class="right">
                {{
                  e.completedAt
                    ? new Date(e.completedAt).getTime() -
                      new Date(e.startedAt).getTime()
                    : ''
                }}ms
              </td>
              <td :title="e.logs">
                <div class="job-log-text">
                  {{ e.logs }}
                </div>
              </td>
              <td :title="e.errorMsg">
                <div class="job-log-text">
                  {{ e.errorMsg }}
                </div>
              </td>
              <td class="center line">
                <SimpleButton
                  v-if="e.status === JobExecutionStatus.Running"
                  :title="$t('p.admin.jobs.abort_execution')"
                  type="danger"
                  small
                  icon="#icon-reject"
                  @click="abortExecution(e)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import {
  getJobDefinitions,
  getJobs,
  deleteJob as deleteJobApi,
  updateJob,
  createJob,
  getJobExecutions,
  cancelJobExecution,
  deleteJobExecutions,
} from '@/api/admin'
import SimpleForm from '@/components/Form'
import {
  FormItem,
  Job,
  JobDefinition,
  JobExecution,
  JobExecutionStatus,
} from '@/types'
import { formatTime, mapOf } from '@/utils'
import { alert, confirm, loading } from '@/utils/ui-utils'
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const jobsList = ref<Job[]>([])

const loadJobsList = async () => {
  try {
    jobsList.value = await getJobs()
  } catch (e: any) {
    alert(e.message)
  }
}

const STATUS_TEXTS = computed(() => ({
  [JobExecutionStatus.Running]: t('p.admin.jobs.running'),
  [JobExecutionStatus.Success]: t('p.admin.jobs.success'),
  [JobExecutionStatus.Failed]: t('p.admin.jobs.failed'),
}))

const edit = ref(false)
const saving = ref(false)
const jobFormEl = ref<InstanceType<typeof SimpleForm>>()
const jobParamsFormEl = ref<InstanceType<typeof SimpleForm>>()
const jobEdit = ref<O>()
const jobParams = ref<O>()
const jobExecutionsShowing = ref<Job>()
const jobExecutions = ref<JobExecution[]>([])

const addJob = () => {
  hideJobExecutions()
  jobEdit.value = {}
  jobParams.value = {}
  edit.value = false
}

const editJob = (job: Job) => {
  hideJobExecutions()
  let params: O
  try {
    params = JSON.parse(job.params)
  } catch {
    alert('invalid params')
    return
  }
  jobEdit.value = {
    id: job.id,
    description: job.description,
    enabled: job.enabled ? '1' : '',
    schedule: job.schedule,
    job: job.job,
  }
  nextTick(() => {
    jobParams.value = params
  })
  edit.value = true
}

const showJobExecutions = async (job: Job) => {
  jobExecutionsShowing.value = job
  loading(true)
  try {
    jobExecutions.value = await getJobExecutions(job.id)
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading()
  }
}

const hideJobExecutions = () => {
  jobExecutionsShowing.value = undefined
  jobExecutions.value = []
}

const cancelEdit = () => {
  jobEdit.value = undefined
  jobParams.value = undefined
  edit.value = false
}

const deleteJob = (job: Job) => {
  confirm({
    title: t('p.admin.jobs.delete_job'),
    message: t('p.admin.jobs.confirm_delete'),
    confirmType: 'danger',
    onOk: () => {
      return deleteJobApi(job.id).then(
        () => {
          if (job.id === jobEdit.value?.id) {
            jobEdit.value = undefined
          }
          loadJobsList()
          if (job.id === jobExecutionsShowing.value?.id) {
            hideJobExecutions()
          }
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const abortExecution = (je: JobExecution) => {
  confirm({
    title: t('p.admin.jobs.abort_execution'),
    message: t('p.admin.jobs.confirm_abort_execution'),
    confirmType: 'danger',
    onOk: () => {
      return cancelJobExecution(je.id).then(
        () => {
          showJobExecutions(jobExecutionsShowing.value!)
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const executionsClearing = ref(false)
const clearExecutions = async () => {
  executionsClearing.value = true
  try {
    await deleteJobExecutions(jobExecutionsShowing.value!.id)
    showJobExecutions(jobExecutionsShowing.value!)
  } catch (e: any) {
    alert(e.message)
  } finally {
    executionsClearing.value = false
  }
}

const saveJob = async () => {
  try {
    await Promise.all([
      jobFormEl.value!.validate(),
      jobParamsFormEl.value!.validate(),
    ])
  } catch {
    return
  }
  const data: Partial<Job> = {
    ...jobEdit.value,
    enabled: !!jobEdit.value!.enabled,
    params: JSON.stringify(jobParams.value!),
  }
  saving.value = true
  try {
    if (edit.value) {
      await updateJob(data.id!, data)
    } else {
      await createJob(data)
    }
    loadJobsList()
    cancelEdit()
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const jobDefinitions = ref<JobDefinition[]>([])
const jobDefinitionsMap = computed(() =>
  mapOf(jobDefinitions.value, (e) => e.name)
)

const jobForm = computed<FormItem[]>(() => [
  {
    field: 'description',
    label: t('p.admin.jobs.desc'),
    type: 'text',
    required: true,
  },
  {
    field: 'enabled',
    label: t('p.admin.jobs.enabled'),
    type: 'checkbox',
    width: '100px',
  },
  {
    field: 'schedule',
    type: 'text',
    label: t('p.admin.jobs.schedule'),
    description: t('p.admin.jobs.schedule_desc'),
    required: true,
  },
  {
    field: 'job',
    label: t('p.admin.jobs.job'),
    type: 'select',
    options: jobDefinitions.value.map((e) => ({
      name: e.displayName,
      title: e.description,
      value: e.name,
    })),
    required: true,
  },
])

const jobParamsForm = computed(() => {
  const job = jobEdit.value?.job
  return jobDefinitionsMap.value[job]?.paramsForm ?? []
})

watch(
  () => jobEdit.value?.job,
  () => {
    jobParams.value = undefined
  }
)

const loadJobDefinitions = async () => {
  try {
    const data = await getJobDefinitions()
    data.forEach((e) => {
      e.paramsForm.forEach((e) => {
        if (e.type === 'form' || e.type === 'code') {
          e.width = '100%'
        }
      })
    })
    jobDefinitions.value = data
  } catch {
    // ignore
  }
}

loadJobsList()
loadJobDefinitions()
</script>
<style lang="scss">
.jobs-manager {
  padding-bottom: 200px;

  .small-title {
    font-size: 18px;
    margin-bottom: 16px;
    display: flex;
    align-items: center;
    overflow: hidden;

    .text {
      flex: 1;
      overflow: hidden;
      white-space: nowrap;
      text-overflow: ellipsis;
    }
  }

  .jobs-list {
    padding: 16px;
  }

  .job-edit {
    padding: 16px;
  }

  .job-executions {
    padding: 16px;
  }

  .job-disabled {
    color: #999;
  }

  .save-button {
    margin-top: 32px;
  }

  .actions {
    margin-bottom: 16px;
  }

  .status-success {
    color: var(--btn-bg-color-success);
  }
  .status-failed {
    color: var(--btn-bg-color-danger);
  }
  .status-running {
    color: var(--btn-bg-color-warning);
  }

  .job-log-text {
    max-width: 190px;
    max-height: 100px;
    white-space: pre;
    overflow: auto;
    font-size: 10px;
    font-family: monospace;
  }
}
</style>
