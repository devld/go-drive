<template>
  <div class="section">
    <h3 class="section-title">
      {{ $t('p.admin.misc.extra_drive') }}
      <SimpleButton :loading="loading" @click="loadData(true)">{{
        $t('p.admin.misc.extra_drive_refresh_repository')
      }}</SimpleButton>
    </h3>
    <div class="extra-drives-table">
      <table class="simple-table">
        <thead>
          <tr>
            <th>{{ $t('p.admin.misc.extra_drive_name') }}</th>
            <th>{{ $t('p.admin.misc.extra_drive_scripts') }}</th>
            <th>{{ $t('p.admin.misc.extra_drive_ops') }}</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="item in data" :key="item.name">
            <tr>
              <td>
                <a
                  class="script-drive-name"
                  :class="{ 'has-description': !!item.description }"
                  href="javascript:;"
                  @click="showScriptDetail(item)"
                  >{{ formatName(item) }}</a
                >
              </td>
              <td>
                <template v-if="item.script">
                  <div class="script-drive-url">
                    <a
                      target="_blank"
                      rel="nofollow noopener noreferrer"
                      :href="item.script.driveUrl"
                      :title="item.script.driveUrl"
                      >{{ item.script.driveUrl }}</a
                    >
                  </div>
                  <div
                    v-if="item.script.driveUploaderUrl"
                    class="script-drive-url"
                  >
                    <a
                      target="_blank"
                      rel="nofollow noopener noreferrer"
                      :href="item.script.driveUploaderUrl"
                      :title="item.script.driveUploaderUrl"
                      >{{ item.script.driveUploaderUrl }}</a
                    >
                  </div>
                </template>
              </td>
              <td class="line">
                <SimpleButton
                  v-if="item.installed"
                  icon="#icon-edit"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.misc.extra_drive_edit')"
                  @click="editDrive(item)"
                />
                <SimpleButton
                  v-if="item.installed"
                  type="danger"
                  icon="#icon-delete"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.misc.extra_drive_uninstall')"
                  @click="doUninstall(item)"
                />
                <SimpleButton
                  v-else
                  icon="#icon-add"
                  :loading="item.loading"
                  :disabled="loading"
                  :title="$t('p.admin.misc.extra_drive_install')"
                  @click="doInstall(item)"
                />
              </td>
            </tr>
            <tr v-if="item.expanded">
              <td colspan="3">
                <div v-markdown="item.description" class="markdown-body"></div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <DialogView v-model:show="edit.showing" fullscreen>
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
import { AvailableDriveScript } from '@/types'
import {
  listAvailableDriveScripts,
  listInstalledDriveScripts,
  installDriveScript,
  uninstallDriveScript,
} from '@/api/admin'
import { alert, confirm } from '@/utils/ui-utils'
import { reactive, ref } from 'vue'
import { mapOf } from '@/utils'
import { useI18n } from 'vue-i18n'

interface DriveScript {
  name: string
  installed: boolean

  loading?: boolean
  expanded?: boolean

  displayName?: string
  description?: string
  script?: AvailableDriveScript
}

const emit = defineEmits<{
  (e: 'timer', v: boolean): void
}>()

const { t } = useI18n()

const loading = ref(false)
const data = ref<DriveScript[]>([])

const edit = reactive({
  showing: false,
  name: '',
})

const loadData = async (force?: boolean) => {
  data.value = []
  loading.value = true
  try {
    const [available, installed] = await Promise.all([
      listAvailableDriveScripts(force),
      listInstalledDriveScripts(),
    ])
    const availableMap = mapOf(available, (e) => e.name)
    const installedMap = mapOf(installed, (e) => e.name)

    const result: DriveScript[] = []

    installed.forEach((e) => {
      result.push({
        name: e.name,
        displayName: e.displayName,
        description: e.description,
        installed: true,
        script: availableMap[e.name],
      })
    })

    available.forEach((e) => {
      if (installedMap[e.name]) return
      result.push({
        name: e.name,
        installed: false,
        script: e,
      })
    })

    data.value = result
  } catch (e: any) {
    alert(e.message)
  } finally {
    loading.value = false
  }
}

const doInstall = async (item: DriveScript) => {
  item.loading = true
  try {
    await installDriveScript(item.script!.name)
    loadData()
  } catch (e: any) {
    alert(e.message)
  } finally {
    item.loading = false
  }
}

const doUninstall = async (item: DriveScript) => {
  try {
    await confirm({
      message: t('p.admin.misc.extra_drive_uninstall_confirm'),
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

const showScriptDetail = (item: DriveScript) => {
  if (item.expanded) {
    item.expanded = false
    return
  }
  if (!item.description) return
  item.expanded = true
}

const formatName = (item: DriveScript) => {
  if (item.displayName) {
    return `${item.displayName} (${item.name})`
  }
  return item.name
}

const editDrive = (item: DriveScript) => {
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
.script-drive-name {
  text-decoration: none;
  color: inherit;

  &.has-description {
    cursor: pointer;
  }
}

.script-drive-url {
  max-width: 40vw;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;

  a {
    text-decoration: none;
    color: inherit;
    color: var(--link-color);
  }
}

.drive-script-editor-wrapper {
  width: 100vw;
  height: 100%;
}
</style>
