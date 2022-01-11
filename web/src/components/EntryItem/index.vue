<template>
  <div
    class="entry-item"
    :class="[
      `entry-item--${entry.type}`,
      `entry-item--view-${viewMode}`,
      entry.type === 'file' ? `entry-item--ext-${ext}` : '',
    ]"
    :title="entryTips"
    @click="emit('click', $event)"
  >
    <span class="entry-item__icon-wrapper">
      <entry-icon
        class="entry-item__icon"
        :entry="entry"
        :icon="icon"
        :show-thumbnail="showThumbnail && viewMode === 'thumbnail'"
        @click="emit('icon-click', $event)"
      />
    </span>
    <span class="entry-item__info">
      <span class="entry-item__name">
        <i v-if="entry.meta.mountAt">@</i>{{ entry.name
        }}<template v-if="entry.meta.ext">.{{ entry.meta.ext }}</template>
      </span>
      <div v-if="viewMode === 'list'" class="entry-item__meta">
        <span class="entry-item__modified-time">{{
          entry.modTime >= 0 ? formatTime(entry.modTime) : ''
        }}</span>
        <span class="entry-item__size">{{
          entry.size >= 0 ? formatBytes(entry.size) : ''
        }}</span>
      </div>
    </span>
  </div>
</template>
<script>
export default { name: 'EntryItem' }
</script>
<script setup>
import { filenameExt, formatBytes, formatTime } from '@/utils'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
  icon: {
    type: String,
  },
  viewMode: {
    type: String,
    default: 'list',
    validator: (val) => val === 'list' || val === 'thumbnail',
  },
  showThumbnail: {
    type: Boolean,
  },
})

const entryTips = computed(() => {
  const entry = props.entry
  if (entry.name === '..') return ''

  let tips =
    entry.name +
    '\n' +
    (entry.type === 'file' ? t('app.file') : t('app.folder')) +
    ' | ' +
    formatBytes(entry.size) +
    '\n' +
    formatTime(entry.modTime)

  if (entry.meta.mountAt) {
    tips += '\n' + t('app.mount_at', { n: entry.meta.mountAt })
  }

  return tips
})

const emit = defineEmits(['click', 'icon-click'])

const ext = computed(() => filenameExt(props.entry.name))
</script>
<style lang="scss">
.entry-item__name {
  i {
    color: #999;
  }
}

.entry-item--view-list {
  display: flex;
  cursor: pointer;
  padding: 4px 16px;

  .entry-item__icon {
    margin-right: 0.5em;
  }

  .entry-item__info {
    flex: 1;
    display: flex;
    align-items: center;
    overflow: hidden;
  }

  .entry-item__name {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .entry-item__meta {
    display: flex;
    align-items: center;
    overflow: hidden;
  }

  .entry-item__modified-time {
    white-space: nowrap;
  }

  .entry-item__size {
    display: block;
    width: 80px;
    text-align: right;
    white-space: nowrap;
  }
}

.entry-item--view-thumbnail {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  padding: 12px;

  .entry-item__icon-wrapper {
    display: block;
    width: 100%;
    padding-top: 100%;
    position: relative;
  }

  .entry-item__icon {
    position: absolute;
    top: 0;
    left: 0;
    bottom: 0;
    right: 0;
    width: unset;
    height: unset;
    margin-bottom: 10px;
  }

  .entry-icon__thumbnail {
    transition: 0.3s;
  }

  &:hover {
    .entry-icon__thumbnail {
      transform: scale(1.2);
    }
  }

  .entry-item__name {
    display: block;
    white-space: nowrap;
    text-align: center;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 14px;
  }
}
</style>
