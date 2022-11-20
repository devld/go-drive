<template>
  <div class="site-config">
    <OptionsConfigure :forms="configForms" />
  </div>
</template>
<script setup lang="ts">
import {
  DEFAULT_AUDIO_FILE_EXTS,
  DEFAULT_IMAGE_FILE_EXTS,
  DEFAULT_TEXT_FILE_EXTS,
  DEFAULT_VIDEO_FILE_EXTS,
} from '@/config'
import { useI18n } from 'vue-i18n'
import OptionsConfigure, { OptionsForm } from './OptionsConfigure.vue'

const { t } = useI18n()

const configForms: OptionsForm[] = [
  {
    title: t('p.admin.site.app_name'),
    defaultOpen: true,
    form: [{ field: 'app.name', type: 'text' }],
  },
  {
    title: t('p.admin.site.file_preview_config'),
    defaultOpen: true,
    form: [
      {
        field: 'web.textFileExts',
        label: t('p.admin.site.text_file_exts'),
        description: t('p.admin.site.text_file_exts_desc'),
        type: 'textarea',
        defaultValue: DEFAULT_TEXT_FILE_EXTS.join(','),
        fillDefaultIfEmpty: true,
      },
      {
        field: 'web.imageFileExts',
        label: t('p.admin.site.image_file_exts'),
        description: t('p.admin.site.image_file_exts_desc'),
        type: 'textarea',
        defaultValue: DEFAULT_IMAGE_FILE_EXTS.join(','),
        fillDefaultIfEmpty: true,
      },
      {
        field: 'web.audioFileExts',
        label: t('p.admin.site.audio_file_exts'),
        description: t('p.admin.site.audio_file_exts_desc'),
        type: 'textarea',
        defaultValue: DEFAULT_AUDIO_FILE_EXTS.join(','),
        fillDefaultIfEmpty: true,
      },
      {
        field: 'web.videoFileExts',
        label: t('p.admin.site.video_file_exts'),
        description: t('p.admin.site.video_file_exts_desc'),
        type: 'textarea',
        defaultValue: DEFAULT_VIDEO_FILE_EXTS.join(','),
        fillDefaultIfEmpty: true,
      },
      {
        field: 'web.monacoEditorExts',
        label: t('p.admin.site.monaco_editor_exts'),
        description: t('p.admin.site.monaco_editor_exts_desc'),
        type: 'textarea',
      },
      {
        field: 'web.officePreviewEnabled',
        label: t('p.admin.site.office_preview_enabled'),
        description: t('p.admin.site.office_preview_enabled_desc'),
        type: 'checkbox',
      },
    ],
  },
  {
    title: t('p.admin.site.anonymous_root_path'),
    form: [
      {
        field: 'anonymous.rootPath',
        description: t('p.admin.site.anonymous_root_path_desc'),
        type: 'text',
      },
    ],
  },
  {
    title: t('p.admin.site.download_options'),
    form: [
      {
        label: t('p.admin.site.proxy_max'),
        field: 'proxy.maxSize',
        description: t('p.admin.site.proxy_max_desc'),
        type: 'text',
      },
      {
        label: t('p.admin.site.zip_max_size'),
        field: 'zip.maxSize',
        description: t('p.admin.site.zip_max_size_desc'),
        type: 'text',
      },
    ],
  },
  {
    title: t('p.admin.site.thumbnail_config'),
    form: [
      {
        field: 'thumbnail.handlersMapping',
        label: t('p.admin.site.thumbnail_mapping'),
        description: t('p.admin.site.thumbnail_mapping_tips'),
        placeholder: t('p.admin.site.thumbnail_mapping_placeholder'),
        type: 'textarea',
        width: '100%',
        validate: (v: string) =>
          !v ||
          !v
            .split('\n')
            .filter(Boolean)
            .some(
              (f) => !/^([A-Za-z0-9-_]+(,[A-Za-z0-9-_]+)*):(.+)$/.test(f)
            ) ||
          t('p.admin.site.thumbnail_mapping_invalid'),
      },
    ],
  },
]
</script>
<style lang="scss">
.site-config {
  padding: 16px;
}
</style>
