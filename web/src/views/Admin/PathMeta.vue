<template>
  <div class="path-meta-manager" :class="{ editing: !!pathMeta }">
    <div v-if="pathMeta" class="path-meta-edit">
      <div class="small-title">
        {{ edit ? $t('p.admin.path_meta.edit') : $t('p.admin.path_meta.add') }}
      </div>
      <div class="path-meta-form">
        <SimpleForm ref="formEl" v-model="pathMeta" :form="pathMetaForm" />
        <div class="save-button">
          <SimpleButton small :loading="saving" @click="savePathMeta">
            {{ $t('p.admin.path_meta.save') }}
          </SimpleButton>
          <SimpleButton small type="info" @click="pathMeta = undefined">
            {{ $t('p.admin.path_meta.cancel') }}
          </SimpleButton>
        </div>
      </div>
    </div>
    <div class="path-meta-list">
      <div v-if="!pathMeta" class="actions">
        <SimpleButton
          icon="#icon-add"
          :title="$t('p.admin.path_meta.add')"
          @click="addPathMeta"
        />
      </div>
      <div class="simple-table-wrapper">
        <table class="simple-table">
          <colgroup>
            <col style="min-width: 100px" />
            <col style="width: 100px" />
            <col style="width: 100px" />
            <col style="width: 100px" />
            <col style="width: 100px" />
            <col style="width: 80px" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ $t('p.admin.path_meta.path') }}</th>
              <th>{{ $t('p.admin.path_meta.password') }}</th>
              <th>{{ $t('p.admin.path_meta.def_sort') }}</th>
              <th>{{ $t('p.admin.path_meta.def_mode') }}</th>
              <th>{{ $t('p.admin.path_meta.hidden_pattern') }}</th>
              <th>{{ $t('p.admin.path_meta.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in pathMetaList" :key="u.path">
              <td class="center">{{ u.path }}</td>
              <td class="center">{{ u.password ? '✅' : '' }}</td>
              <td class="center">
                {{ u.defaultSort ? $t(sortingNamesMap[u.defaultSort]) : '' }}
              </td>
              <td class="center">
                {{ u.defaultMode ? $t(listModesMap[u.defaultMode]) : '' }}
              </td>
              <td class="center">{{ u.hiddenPattern ? '✅' : '' }}</td>
              <td class="center line">
                <SimpleButton
                  :title="$t('p.admin.path_meta.edit')"
                  small
                  icon="#icon-edit"
                  @click="editPathMeta(u)"
                />
                <SimpleButton
                  :title="$t('p.admin.path_meta.delete')"
                  type="danger"
                  small
                  icon="#icon-delete"
                  @click="deletePathMeta(u)"
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
  getAllPathMeta,
  savePathMeta as savePathMetaApi,
  deletePathMeta as deletePathMetaApi,
} from '@/api/admin'
import { sortModes } from '@/components/entry/sort'
import { FormItem, PathMeta } from '@/types'
import { mapOf } from '@/utils'
import { alert, confirm } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const pathMetaList = ref<PathMeta[]>([])

const pathMeta = ref<O>()
const edit = ref(false)
const saving = ref(false)

const formEl = ref<InstanceType<SimpleFormType> | null>(null)

const pathMetaForm = computed<FormItem[]>(() => [
  {
    field: 'path',
    label: t('p.admin.path_meta.f_path'),
    type: 'text',
    width: '100%',
  },
  {
    field: 'password',
    label: t('p.admin.path_meta.f_password'),
    description: t('p.admin.path_meta.f_password_desc'),
    type: 'text',
    width: '50%',
  },
  {
    field: 'passwordR',
    label: t('p.admin.path_meta.f_password_r'),
    type: 'checkbox',
    width: '50%',
  },
  {
    field: 'defaultSort',
    label: t('p.admin.path_meta.f_def_sort'),
    type: 'select',
    options: [
      { name: '', value: '' },
      ...sortModes.map((e) => ({ name: t(e.name), value: e.key })),
    ],
    width: '50%',
  },
  {
    field: 'defaultSortR',
    label: t('p.admin.path_meta.f_def_sort_r'),
    type: 'checkbox',
    width: '50%',
  },
  {
    field: 'defaultMode',
    label: t('p.admin.path_meta.f_def_mode'),
    type: 'select',
    options: [
      { name: '', value: '' },
      { name: t('p.admin.path_meta.fo_mode_list'), value: 'list' },
      { name: t('p.admin.path_meta.fo_mode_thumbnail'), value: 'thumbnail' },
    ],
    width: '50%',
  },
  {
    field: 'defaultModeR',
    label: t('p.admin.path_meta.f_def_mode_r'),
    type: 'checkbox',
    width: '50%',
  },
  {
    field: 'hiddenPattern',
    label: t('p.admin.path_meta.f_hidden_pattern'),
    description: t('p.admin.path_meta.f_hidden_pattern_desc'),
    type: 'text',
    width: '50%',
  },
  {
    field: 'hiddenPatternR',
    label: t('p.admin.path_meta.f_hidden_pattern_r'),
    type: 'checkbox',
    width: '50%',
  },
])

const sortingNamesMap = mapOf(
  sortModes,
  (e) => e.key,
  (e) => e.name
)
const listModesMap: O<string> = {
  list: 'p.admin.path_meta.fo_mode_list',
  thumbnail: 'p.admin.path_meta.fo_mode_thumbnail',
}

const loadPathMetaList = async () => {
  try {
    pathMetaList.value = await getAllPathMeta()
  } catch (e: any) {
    alert(e.message)
  }
}

const addPathMeta = () => {
  pathMeta.value = {
    path: '',
    password: '',
    defaultSort: '',
    defaultMode: '',
    hiddenPattern: '',

    passwordR: '',
    defaultSortR: '',
    defaultModeR: '',
    hiddenPatternR: '',
  }
  edit.value = false
}

const recursiveFields = [
  'passwordR',
  'defaultSortR',
  'defaultModeR',
  'hiddenPatternR',
]

const editPathMeta = async (pathMeta_: PathMeta) => {
  pathMeta.value = { ...pathMeta_ }
  recursiveFields.forEach((f, i) => {
    pathMeta.value![f] = pathMeta_.recursive & (1 << i) ? '1' : ''
  })
  edit.value = true
}

const deletePathMeta = async (pathMeta_: PathMeta) => {
  confirm({
    title: t('p.admin.path_meta.delete_item'),
    message: t('p.admin.path_meta.confirm_delete'),
    confirmType: 'danger',
    onOk: () => {
      return deletePathMetaApi(pathMeta_.path).then(
        () => {
          if (pathMeta_.path === pathMeta.value?.path) {
            pathMeta.value = undefined
          }
          loadPathMetaList()
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const savePathMeta = async () => {
  try {
    await formEl.value!.validate()
  } catch {
    return
  }
  const v = pathMeta.value!

  const data = { ...pathMeta.value, recursive: 0 } as O
  recursiveFields.forEach((f, i) => {
    data.recursive |= +!!data[f] << i
    delete data[f]
  })

  saving.value = true
  try {
    await savePathMetaApi(v.path, data)
    loadPathMetaList()
    pathMeta.value = undefined
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

loadPathMetaList()
</script>
<style lang="scss">
.path-meta-manager {
  .path-meta-edit {
    max-width: 500px;
    padding: 16px;
  }

  .path-meta-list {
    padding: 16px;
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
  }

  .simple-form {
    margin-bottom: 10px;
  }

  .save-button {
    margin-top: 32px;
  }
}
</style>
