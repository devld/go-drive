<template>
  <div class="drives-manager" :class="{ editing: !!drive }">
    <div class="drives-list">
      <div class="actions">
        <SimpleButton
          class="add-button"
          icon="#icon-add"
          :title="$t('p.admin.drive.add_drive')"
          @click="addDrive"
        />
        <SimpleButton
          icon="#icon-refresh2"
          :title="$t('p.admin.drive.reload_tip')"
          :loading="reloading"
          @click="reloadDrives"
        >
          {{ $t('p.admin.drive.reload_drives') }}
        </SimpleButton>
      </div>
      <table class="simple-table">
        <thead>
          <tr>
            <th>{{ $t('p.admin.drive.name') }}</th>
            <th>{{ $t('p.admin.drive.type') }}</th>
            <th>{{ $t('p.admin.drive.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="d in drives"
            :key="d.name"
            :class="{ 'not-enabled-drive': !d.enabled }"
          >
            <td class="center">{{ d.name }}</td>
            <td class="center">{{ d.type }}</td>
            <td class="center line">
              <SimpleButton
                :title="$t('p.admin.drive.edit')"
                small
                icon="#icon-edit"
                @click="editDrive(d)"
              />
              <SimpleButton
                :title="$t('p.admin.drive.delete')"
                type="danger"
                small
                icon="#icon-delete"
                @click="deleteDrive(d)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="drive" class="drive-edit">
      <div class="small-title">
        {{
          edit
            ? $t('p.admin.drive.edit_drive', { n: drive.name })
            : $t('p.admin.drive.add_drive')
        }}
      </div>

      <div class="drive-form">
        <SimpleForm
          ref="baseFormEl"
          v-model="drive"
          :form="baseForm"
          no-auto-complete
        />

        <template v-if="drive.type && driveFactoriesMap[drive.type]">
          <details
            v-if="driveFactoriesMap[drive.type].readme"
            class="drive-config-readme"
          >
            <summary>
              {{ driveFactoriesMap[drive.type].displayName }} README
            </summary>
            <div
              v-markdown="driveFactoriesMap[drive.type].readme"
              class="markdown-body"
            ></div>
          </details>

          <SimpleForm
            ref="configFormEl"
            :key="drive.type"
            v-model="drive.config"
            no-auto-complete
            :form="driveFactoriesMap[drive.type].configForm"
          />
        </template>

        <div class="save-button">
          <SimpleButton small :loading="saving" @click="saveDrive">
            {{ $t('p.admin.drive.save') }}
          </SimpleButton>
          <SimpleButton small type="info" @click="cancelEdit">
            {{ $t('p.admin.drive.cancel') }}
          </SimpleButton>
        </div>
      </div>
      <div v-if="drive && driveInit" class="drive-init">
        <div class="small-title">
          {{ $t('p.admin.drive.configure') }}
          <span
            class="drive-init-state"
            :class="{ 'drive-configured': driveInit.configured }"
            >{{
              driveInit.configured
                ? $t('p.admin.drive.configured')
                : $t('p.admin.drive.not_configured')
            }}</span
          >
        </div>
        <OAuthConfigure
          v-if="driveInit.oauth"
          :key="drive.name"
          :configured="driveInit.configured"
          :data="driveInit.oauth"
          :drive="drive"
          @refresh="getDriveInitConfigInfo"
        />
        <div v-if="driveInit.form" class="drive-init-form">
          <SimpleForm
            ref="initFormEl"
            v-model="driveInitForm"
            :form="driveInit.form"
          />
          <SimpleButton small @click="saveDriveConfig">
            {{ $t('p.admin.drive.start_configure') }}
          </SimpleButton>
        </div>
      </div>
    </div>
    <div v-else class="edit-tips">
      <SimpleButton icon="#icon-add" title="Add drive" small @click="addDrive">
        {{ $t('p.admin.drive.add') }}
      </SimpleButton>
      {{ $t('p.admin.drive.or_edit') }}
    </div>
  </div>
</template>
<script setup lang="ts">
import {
  createDrive,
  deleteDrive as deleteDriveApi,
  getDrives,
  reloadDrives as reloadDrivesApi,
  updateDrive,
  getDriveInitConfig,
  initDrive,
  getDriveFactories,
} from '@/api/admin'
import { alert, confirm, loading } from '@/utils/ui-utils'

import OAuthConfigure from './drive-configure/OAuth.vue'
import { mapOf } from '@/utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Drive, DriveFactoryConfig, DriveInitConfig, FormItem } from '@/types'

const { t } = useI18n()

const drives = ref<Drive[]>([])
const drive = ref<O | null>(null)
const edit = ref(false)
const saving = ref(false)
const driveInit = ref<DriveInitConfig | null>(null)
const driveInitForm = ref<O<string>>({})
const reloading = ref(false)
const driveFactories = ref<DriveFactoryConfig[]>([])

const driveFactoriesMap = computed(() =>
  mapOf(driveFactories.value, (f) => f.type)
)

const baseFormEl = ref<InstanceType<SimpleFormType> | null>(null)
const configFormEl = ref<InstanceType<SimpleFormType> | null>(null)
const initFormEl = ref<InstanceType<SimpleFormType> | null>(null)

const baseForm = computed<FormItem[]>(() => [
  {
    field: 'name',
    label: t('p.admin.drive.f_name'),
    type: 'text',
    required: true,
    disabled: edit.value,
  },
  {
    field: 'enabled',
    label: t('p.admin.drive.f_enabled'),
    type: 'checkbox',
  },
  {
    field: 'type',
    label: t('p.admin.drive.f_type'),
    type: 'select',
    required: true,
    disabled: edit.value,
    options: driveFactories.value.map((f) => ({
      name: f.displayName,
      value: f.type,
    })),
  },
])

const showReloadingTips = () => {
  if (!localStorage.getItem('drive-reloading-tips')) {
    alert(t('p.admin.drive.reload_tips'))
    localStorage.setItem('drive-reloading-tips', '1')
  }
}

const loadDrives = async () => {
  try {
    driveFactories.value = await getDriveFactories()
    drives.value = await getDrives()
  } catch (e: any) {
    alert(e.message)
  }
}

const addDrive = () => {
  drive.value = {
    name: '',
    enabled: '1',
    type: '',
    config: '',
  }
  edit.value = false
}

const editDrive = (drive_: Drive) => {
  drive.value = {
    name: drive_.name,
    enabled: drive_.enabled ? '1' : '',
    type: drive_.type,
    config: JSON.parse(drive_.config),
  }
  edit.value = true
  getDriveInitConfigInfo()
}

const deleteDrive = (drive_: Drive) => {
  confirm({
    title: t('p.admin.drive.delete_drive'),
    message: t('p.admin.drive.confirm_delete', { n: drive_.name }),
    confirmType: 'danger',
    onOk: () => {
      return deleteDriveApi(drive_.name).then(
        () => {
          if (drive_.name === drive.value?.name) {
            drive.value = null
          }
          loadDrives()
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const saveDrive = async () => {
  try {
    await Promise.all([
      baseFormEl.value?.validate(),
      configFormEl.value?.validate(),
    ])
  } catch {
    return
  }

  const d = {
    name: drive.value!.name,
    enabled: !!drive.value!.enabled,
    type: drive.value!.type,
    config: JSON.stringify(drive.value!.config),
  }
  saving.value = true
  try {
    if (edit.value) {
      await updateDrive(drive.value!.name, d)
    } else {
      await createDrive(d)
    }
    edit.value = true

    showReloadingTips()
  } catch (e: any) {
    alert(e.message)
    return
  } finally {
    saving.value = false
  }
  getDriveInitConfigInfo()
  loadDrives()
}

const cancelEdit = () => {
  drive.value = null
  driveInit.value = null
  driveInitForm.value = {}
}

const getDriveInitConfigInfo = async () => {
  loading(true)
  try {
    driveInit.value = await getDriveInitConfig(drive.value!.name)
    driveInitForm.value = driveInit.value?.value || {}
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading()
  }
}

const saveDriveConfig = async () => {
  try {
    await initFormEl.value!.validate()
  } catch {
    return
  }
  loading(true)
  try {
    await initDrive(drive.value!.name, driveInitForm.value)
    showReloadingTips()
  } catch (e: any) {
    alert(e.message)
    return
  } finally {
    loading()
  }
  getDriveInitConfigInfo()
}

const reloadDrives = async () => {
  reloading.value = true
  try {
    await reloadDrivesApi()
  } catch (e: any) {
    alert(e.message)
  } finally {
    reloading.value = false
  }
}

loadDrives()
</script>
<style lang="scss">
.drives-manager {
  display: flex;

  .drive-edit {
    padding: 16px;
  }

  .drives-list {
    padding: 16px;
  }

  .not-enabled-drive {
    color: #999;
  }

  .drive-config-readme {
    margin: 1em 0 2em;

    .markdown-body {
      margin-top: 1em;
    }
  }

  .drive-init {
    margin-top: 32px;
  }

  .drive-init-state {
    margin-left: 1em;
    font-size: 14px;
    color: #ffa000;
  }

  .drive-configured {
    color: #00e676;
  }

  .drive-init-form {
    margin-top: 1em;
  }

  .small-title {
    font-size: 18px;
    margin-bottom: 16px;
  }

  .edit-tips {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    white-space: pre;
  }

  .actions {
    margin-bottom: 16px;

    .add-button {
      display: none;
    }
  }

  .user-item {
    &:not(:last-child) {
      margin-right: 10px;
    }
  }

  .save-button {
    margin-top: 32px;
  }

  .drive-form {
    .simple-form:first-child {
      margin-bottom: 10px;
    }
  }

  @media screen and (max-width: 600px) {
    justify-content: center;

    .actions {
      .add-button {
        display: inline;
      }
    }

    .edit-tips {
      display: none;
    }

    &.editing {
      .edit-tips {
        display: block;
      }

      .drives-list {
        display: none;
      }
    }
  }
}
</style>
