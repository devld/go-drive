<template>
  <div class="extra-drives-manager">
    <div class="page-actions">
      <SimpleButton :loading="loading" @click="loadData(true)">
        {{ $t('p.admin.extra_drive.refresh_repository') }}
        <template v-if="syncProgress" #loading>{{ syncProgress }}</template>
      </SimpleButton>
    </div>
    <div class="extra-drives-table">
      <table class="simple-table full-width">
        <colgroup>
          <col style="min-width: 100px" />
          <col style="min-width: 100px" />
          <col style="width: 140px" />
        </colgroup>
        <thead>
          <tr>
            <th>{{ $t('p.admin.extra_drive.name') }}</th>
            <th>{{ $t('p.admin.extra_drive.scripts') }}</th>
            <th>{{ $t('p.admin.extra_drive.ops') }}</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="item in data" :key="item.name">
            <tr>
              <td>
                <a
                  class="script-drive-name"
                  :class="{ 'has-description': !!scriptDescription(item) }"
                  href="javascript:;"
                  @click="showScriptDetail(item)"
                  >{{ formatName(item) }}</a
                >
                <div v-if="versionText(item)" class="script-drive-version">
                  {{ versionText(item) }}
                </div>
              </td>
              <td>
                <template v-if="item.driveUrl">
                  <div class="script-drive-url">
                    <a
                      target="_blank"
                      rel="nofollow noopener noreferrer"
                      :href="item.driveUrl"
                      :title="item.driveUrl"
                      >{{ item.driveUrl }}</a
                    >
                  </div>
                  <div
                    v-if="item.driveUploaderUrl"
                    class="script-drive-url"
                  >
                    <a
                      target="_blank"
                      rel="nofollow noopener noreferrer"
                      :href="item.driveUploaderUrl"
                      :title="item.driveUploaderUrl"
                      >{{ item.driveUploaderUrl }}</a
                    >
                  </div>
                </template>
              </td>
              <td class="line right">
                <SimpleButton
                  v-if="item.updateAvailable"
                  icon="refresh"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.extra_drive.update')"
                  @click="doInstall(item)"
                />
                <SimpleButton
                  v-if="item.installed"
                  icon="edit"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.extra_drive.edit')"
                  @click="editDrive(item)"
                />
                <SimpleButton
                  v-if="item.installed"
                  type="danger"
                  icon="delete"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.extra_drive.uninstall')"
                  @click="doUninstall(item)"
                />
                <SimpleButton
                  v-else
                  icon="add"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.extra_drive.install')"
                  @click="doInstall(item)"
                />
              </td>
            </tr>
            <tr v-if="item.expanded">
              <td colspan="3">
                <div
                  v-markdown="scriptDescription(item)"
                  class="markdown-body"
                  data-ui="markdown"
                ></div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <DialogView
      v-model:show="edit.showing"
      fullscreen
      :accessible-label="edit.name || $t('routes.title.extra_drives')"
    >
      <div class="drive-script-editor-wrapper">
        <DriveCodeEditor
          v-if="edit.name"
          :key="edit.name"
          :name="edit.name"
          @close="onScriptEditClose"
        />
      </div>
    </DialogView>
  </div>
</template>
<script lang="ts" setup>
import DriveCodeEditor from './DriveCodeEditor.vue'
import { DriveScript } from '@/types'
import {
  listDriveScripts,
  installDriveScript,
  uninstallDriveScript,
  syncDriveScriptsRepository,
} from '@/api/admin'
import { alert, confirm } from '@/utils/ui-utils'
import { reactive, ref } from 'vue'
import { taskDone } from '@/utils'
import { useI18n } from 'vue-i18n'

interface DriveScriptRow extends DriveScript {
  loading?: boolean
  expanded?: boolean
}

const emit = defineEmits<{
  (e: 'timer', v: boolean): void
}>()

const { t } = useI18n()

const loading = ref(false)
const data = ref<DriveScriptRow[]>([])
const syncProgress = ref('')

const edit = reactive({
  showing: false,
  name: '',
})

const syncRepository = async () => {
  syncProgress.value = ''
  await taskDone(syncDriveScriptsRepository(), (task) => {
    const loaded = task.progress?.loaded || 0
    const total = task.progress?.total || 0
    syncProgress.value = t('p.admin.extra_drive.sync_progress', {
      loaded,
      total: total || '-',
    })
  })
  syncProgress.value = ''
}

const loadData = async (force?: boolean) => {
  loading.value = true
  try {
    let repo = await listDriveScripts()
    if (force || !repo.ready) {
      await syncRepository()
      repo = await listDriveScripts()
    }
    data.value = repo.scripts || []
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading.value = false
    syncProgress.value = ''
  }
}

const doInstall = async (item: DriveScriptRow) => {
  item.loading = true
  try {
    await installDriveScript(item.name)
    await loadData()
  } catch (e: any) {
    alert(e.message)
  } finally {
    item.loading = false
  }
}

const doUninstall = async (item: DriveScriptRow) => {
  try {
    await confirm({
      message: t('p.admin.extra_drive.uninstall_confirm'),
      confirmType: 'danger',
    })
  } catch {
    return
  }
  item.loading = true
  try {
    await uninstallDriveScript(item.name)
    loadData()
  } catch (e: any) {
    alert(e.message)
  } finally {
    item.loading = false
  }
}

const showScriptDetail = (item: DriveScriptRow) => {
  if (item.expanded) {
    item.expanded = false
    return
  }
  if (!scriptDescription(item)) return
  item.expanded = true
}

const scriptDescription = (item: DriveScriptRow) => {
  return item.installed?.description || item.description
}

const formatName = (item: DriveScriptRow) => {
  const displayName = item.installed?.displayName || item.displayName
  if (displayName && displayName !== item.name) {
    return `${displayName} (${item.name})`
  }
  return item.name
}

const versionText = (item: DriveScriptRow) => {
  if (item.updateAvailable) {
    return `v${item.installed?.version || '?'} → v${item.version}`
  }
  const version = item.installed?.version || item.version
  return version ? `v${version}` : ''
}

const editDrive = (item: DriveScriptRow) => {
  edit.name = item.name
  edit.showing = true
  emit('timer', false)
}

const onScriptEditClose = () => {
  edit.showing = false
  edit.name = ''
  emit('timer', true)
}

loadData()
</script>
<style lang="scss">
.extra-drives-manager {
  padding: 16px;

  .page-actions {
    margin-bottom: 16px;
  }

  .script-drive-name {
    text-decoration: none;
    color: inherit;

    &.has-description {
      cursor: pointer;
    }
  }

  .script-drive-version {
    color: var(--color-text-secondary);
    font-size: 0.85em;
  }

  .script-drive-url {
    max-width: 40vw;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;

    a {
      text-decoration: none;
      color: inherit;
      color: var(--color-accent);
    }
  }

  .extra-drives-table .markdown-body {
    background-color: transparent;
  }
}

.drive-script-editor-wrapper {
  width: 100%;
  height: 100%;
  min-height: 0;
  align-self: stretch;
}
</style>
