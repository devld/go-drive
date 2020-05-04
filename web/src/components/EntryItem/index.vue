<template>
  <div
    class="entry-item"
    :class="[`entry-item--${entry.type}`, `entry-item--ext-${ext}`]"
    @click="$emit('click', entry)"
  >
    <span class="entry-item__icon">
      <svg class="icon" aria-hidden="true">
        <use :xlink:href="entryIcon" />
      </svg>
    </span>
    <span class="entry-item__info">
      <span class="entry-item__name">{{ entry.name }}</span>
      <span
        class="entry-item__modified-time"
      >{{ entry.updated_at > 0 ? $.formatTime(entry.updated_at) : '' }}</span>
      <span class="entry-item__size">{{ entry.size > 0 ? $.formatBytes(entry.size) : '' }}</span>
    </span>
  </div>
</template>
<script>
import { filenameExt } from '@/utils'
import { getIconSVG } from './file-icon'

export default {
  name: 'EntryItem',
  props: {
    entry: {
      type: Object,
      required: true
    }
  },
  computed: {
    ext () {
      return filenameExt(this.entry.name)
    },
    entryIcon () {
      return getIconSVG(this.entry)
    }
  }
}
</script>
<style lang="scss">
.entry-item {
  display: flex;
  cursor: pointer;
  padding: 4px 16px;

  &:hover {
    background-color: rgba(0, 0, 0, 0.08);
  }
}

.entry-item__icon {
  margin-right: 0.5em;

  .icon {
    font-size: 42px;
  }
}

.entry-item__info {
  flex: 1;
  display: flex;
  align-items: center;
}

.entry-item__name {
  font-size: 16px;
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.entry-item__modified-time {
  font-size: 14px;
  white-space: nowrap;
}

.entry-item__size {
  width: 100px;
  font-size: 14px;
  text-align: right;
  white-space: nowrap;
}
</style>
