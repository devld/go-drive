<template>
  <div class="file-bucket-manager" :class="{ editing: !!fileBucket }">
    <div v-if="fileBucket" class="file-bucket-edit">
      <div class="small-title">
        {{
          edit ? $t('p.admin.file_bucket.edit') : $t('p.admin.file_bucket.add')
        }}
      </div>
      <div class="file-bucket-form">
        <SimpleForm ref="formEl" v-model="fileBucket" :form="pathMetaForm" />
        <div class="save-button">
          <SimpleButton small :loading="saving" @click="saveFileBucket">
            {{ $t('p.admin.file_bucket.save') }}
          </SimpleButton>
          <SimpleButton small type="info" @click="fileBucket = undefined">
            {{ $t('p.admin.file_bucket.cancel') }}
          </SimpleButton>
        </div>
      </div>
    </div>
    <div
      v-if="fileBucket?.name"
      v-markdown="
        $t('p.admin.file_bucket.upload_help_doc_md', {
          api: uploadURLDoc(false),
          api_with_path: uploadURLDoc(true),
        })
      "
      class="file-bucket-upload-help markdown-body"
    ></div>
    <div class="file-bucket-list">
      <div v-if="!fileBucket" class="actions">
        <SimpleButton
          icon="#icon-add"
          :title="$t('p.admin.file_bucket.add')"
          @click="addFileBucket"
        />
      </div>
      <div class="simple-table-wrapper">
        <table class="simple-table">
          <colgroup>
            <col style="width: 20%; min-width: 100px" />
            <col style="min-width: 200px" />
            <col style="width: 80px" />
          </colgroup>
          <thead>
            <tr>
              <th>{{ $t('p.admin.file_bucket.name') }}</th>
              <th>{{ $t('p.admin.file_bucket.target_path') }}</th>
              <th>{{ $t('p.admin.file_bucket.operation') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in fileBuckets" :key="u.name">
              <td class="center">{{ u.name }}</td>
              <td>{{ u.targetPath }}</td>
              <td class="center line">
                <SimpleButton
                  :title="$t('p.admin.file_bucket.edit')"
                  small
                  icon="#icon-edit"
                  @click="editFileBucket(u)"
                />
                <SimpleButton
                  :title="$t('p.admin.file_bucket.delete')"
                  type="danger"
                  small
                  icon="#icon-delete"
                  @click="deleteFileBucket(u)"
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
  getAllFileBuckets,
  createFileBucket,
  updateFileBucket,
  deleteFileBucket as deletePathMetaApi,
} from '@/api/admin'
import { API_PATH } from '@/api/http'
import { FormItem, FileBucket } from '@/types'
import { alert, confirm } from '@/utils/ui-utils'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { FILE_BUCKET_SECRET_KEY } from '@/config'

const { t } = useI18n()

const fileBuckets = ref<FileBucket[]>([])

const fileBucket = ref<O>()
const edit = ref(false)
const saving = ref(false)

const formEl = ref<InstanceType<SimpleFormType> | null>(null)

const pathMetaForm = computed<FormItem[]>(() => [
  {
    field: 'name',
    label: t('p.admin.file_bucket.f_name'),
    description: t('p.admin.file_bucket.f_name_desc'),
    type: 'text',
    width: '100%',
    required: true,
    disabled: edit.value,
  },
  {
    field: 'targetPath',
    label: t('p.admin.file_bucket.f_target_path'),
    description: t('p.admin.file_bucket.f_target_path_desc'),
    type: 'text',
    width: '100%',
    required: true,
  },
  {
    field: 'keyTemplate',
    label: t('p.admin.file_bucket.f_key_template'),
    description: t('p.admin.file_bucket.f_key_template_desc'),
    type: 'text',
    width: '100%',
  },
  {
    field: 'secretToken',
    label: t('p.admin.file_bucket.f_secret_token'),
    description: t('p.admin.file_bucket.f_secret_token_desc'),
    type: 'password',
    width: '100%',
    required: true,
  },
  {
    field: 'urlTemplate',
    label: t('p.admin.file_bucket.f_url_template'),
    description: t('p.admin.file_bucket.f_url_template_desc'),
    type: 'text',
    width: '100%',
  },
  {
    field: 'customKey',
    label: t('p.admin.file_bucket.f_custom_key'),
    description: t('p.admin.file_bucket.f_custom_key_desc'),
    type: 'checkbox',
    width: '50%',
  },
  {
    field: 'maxSize',
    label: t('p.admin.file_bucket.f_max_size'),
    description: t('p.admin.file_bucket.f_max_size_desc'),
    type: 'text',
    width: '50%',
  },
  {
    field: 'allowedTypes',
    label: t('p.admin.file_bucket.f_allowed_types'),
    description: t('p.admin.file_bucket.f_allowed_types_desc'),
    type: 'text',
    width: '100%',
  },
  {
    field: 'allowedReferrers',
    label: t('p.admin.file_bucket.f_allowed_referrers'),
    description: t('p.admin.file_bucket.f_allowed_referrers_desc'),
    type: 'text',
    width: '100%',
  },
  {
    field: 'cacheMaxAge',
    label: t('p.admin.file_bucket.f_cache_max_age'),
    description: t('p.admin.file_bucket.f_cache_max_age_desc'),
    placeholder: '1d',
    type: 'text',
    width: '100%',
  },
])

const loadFileBuckets = async () => {
  try {
    fileBuckets.value = await getAllFileBuckets()
  } catch (e: any) {
    alert(e.message)
  }
}

const addFileBucket = () => {
  fileBucket.value = {}
  edit.value = false
}

const editFileBucket = (fileBucket_: FileBucket) => {
  fileBucket.value = {
    ...fileBucket_,
    customKey: fileBucket_.customKey ? '1' : '',
  }
  edit.value = true
}

const deleteFileBucket = async (fileBucket_: FileBucket) => {
  confirm({
    title: t('p.admin.file_bucket.delete_item'),
    message: t('p.admin.file_bucket.confirm_delete'),
    confirmType: 'danger',
    onOk: () => {
      return deletePathMetaApi(fileBucket_.name).then(
        () => {
          if (fileBucket_.name === fileBucket.value?.name) {
            fileBucket.value = undefined
          }
          loadFileBuckets()
        },
        (e) => {
          alert(e.message)
          return Promise.reject(e)
        }
      )
    },
  })
}

const saveFileBucket = async () => {
  try {
    await formEl.value!.validate()
  } catch {
    return
  }
  const v = fileBucket.value!
  v.customKey = !!v.customKey
  saving.value = true
  try {
    if (edit.value) await updateFileBucket(v.name, v)
    else await createFileBucket(v)
    loadFileBuckets()
    fileBucket.value = undefined
  } catch (e: any) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

const uploadURLDoc = (withPath: boolean) => {
  const fb = fileBucket.value
  if (!fb?.name) return
  let url = `${API_PATH}/f/${encodeURIComponent(fb.name)}`
  if (withPath) url += `/${t('p.admin.file_bucket.upload_api_p_path')}`
  return (
    url +
    `?${FILE_BUCKET_SECRET_KEY}=${t(
      'p.admin.file_bucket.upload_api_p_secret_token'
    )}`
  )
}

loadFileBuckets()
</script>
<style lang="scss">
.file-bucket-manager {
  .file-bucket-edit {
    max-width: 500px;
    padding: 16px;
  }

  .file-bucket-list {
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

.file-bucket-upload-help {
  margin-top: 20px;
  padding: 16px;
}
</style>
