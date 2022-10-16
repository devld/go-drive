<template>
  <div class="misc-settings">
    <RootPermissions />
    <OptionsConfigure
      :title="$t('p.admin.misc.file_preview_config')"
      :form="handlerExtsForm"
    />
    <OptionsConfigure
      :title="$t('p.admin.misc.anonymous_root_path')"
      :form="anonymousRootPathForm"
    />
    <OptionsConfigure
      :title="$t('p.admin.misc.thumbnail_config')"
      :form="thumbnailForm"
    />
    <OptionsConfigure
      :title="$t('p.admin.misc.proxy_max')"
      :form="proxyMaxForm"
    />
    <SearchIndex />
    <CleanInvalid />
    <CleanCache />
    <SysStats />
  </div>
</template>
<script lang="ts">
export default { name: 'MiscSettings' }
</script>
<script setup lang="ts">
import RootPermissions from './RootPermissions.vue'
import CleanInvalid from './CleanInvalid.vue'
import CleanCache from './CleanCache.vue'
import SysStats from './SysStats.vue'
import SearchIndex from './SearchIndex.vue'
import OptionsConfigure from '../OptionsConfigure.vue'
import { ref } from 'vue'
import { FormItem } from '@/types'
import { useI18n } from 'vue-i18n'
import {
  DEFAULT_IMAGE_FILE_EXTS,
  DEFAULT_VIDEO_FILE_EXTS,
  DEFAULT_TEXT_FILE_EXTS,
  DEFAULT_AUDIO_FILE_EXTS,
} from '@/config'

const { t } = useI18n()

const thumbnailForm = ref<FormItem[]>([
  {
    field: 'thumbnail.handlersMapping',
    label: t('p.admin.misc.thumbnail_mapping'),
    description: t('p.admin.misc.thumbnail_mapping_tips'),
    placeholder: t('p.admin.misc.thumbnail_mapping_placeholder'),
    type: 'textarea',
    width: '100%',
    validate: (v: string) =>
      !v ||
      !v
        .split('\n')
        .filter(Boolean)
        .some((f) => !/^([A-Za-z0-9-_]+(,[A-Za-z0-9-_]+)*):(.+)$/.test(f)) ||
      t('p.admin.misc.thumbnail_mapping_invalid'),
  },
])

const handlerExtsForm = ref<FormItem[]>([
  {
    field: 'web.textFileExts',
    label: t('p.admin.misc.text_file_exts'),
    description: t('p.admin.misc.text_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_TEXT_FILE_EXTS.join(','),
    fillDefaultIfEmpty: true,
  },
  {
    field: 'web.imageFileExts',
    label: t('p.admin.misc.image_file_exts'),
    description: t('p.admin.misc.image_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_IMAGE_FILE_EXTS.join(','),
    fillDefaultIfEmpty: true,
  },
  {
    field: 'web.audioFileExts',
    label: t('p.admin.misc.audio_file_exts'),
    description: t('p.admin.misc.audio_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_AUDIO_FILE_EXTS.join(','),
    fillDefaultIfEmpty: true,
  },
  {
    field: 'web.videoFileExts',
    label: t('p.admin.misc.video_file_exts'),
    description: t('p.admin.misc.video_file_exts_desc'),
    type: 'textarea',
    defaultValue: DEFAULT_VIDEO_FILE_EXTS.join(','),
    fillDefaultIfEmpty: true,
  },
  {
    field: 'web.officePreviewEnabled',
    label: t('p.admin.misc.office_preview_enabled'),
    description: t('p.admin.misc.office_preview_enabled_desc'),
    type: 'checkbox',
  },
])

const anonymousRootPathForm = ref<FormItem[]>([
  {
    field: 'anonymous.rootPath',
    description: t('p.admin.misc.anonymous_root_path_desc'),
    type: 'text',
  },
])

const proxyMaxForm = ref<FormItem[]>([
  {
    field: 'proxy.maxSize',
    description: t('p.admin.misc.proxy_max_desc'),
    type: 'text',
  },
])
</script>
<style lang="scss">
.misc-settings {
  padding: 16px;

  .section {
    padding-top: 1em;
    margin-bottom: 2em;

    &:not(:first-child) {
      border-top: solid 1px;
      border-color: var(--border-color);
    }
  }

  .section-title {
    margin: 0 0 16px;
    font-size: 20px;
    font-weight: normal;
  }
}
</style>
