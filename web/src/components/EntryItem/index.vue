<template>
  <div
    class="entry-item"
    :class="[
      `entry-item--${entry.type}`,
      entry.type === 'file' ? `entry-item--ext-${ext}` : '',
    ]"
    @click="$emit('click', $event)"
  >
    <i-icon
      class="entry-item__icon"
      v-if="icon"
      :svg="icon"
      @click="$emit('icon-click', $event)"
    />
    <entry-icon
      v-else
      class="entry-item__icon"
      :entry="entry"
      @click="$emit('icon-click', $event)"
    />
    <span class="entry-item__info">
      <span class="entry-item__name">{{ entry.name }}</span>
      <span class="entry-item__modified-time">{{
        entry.mod_time >= 0 ? $.formatTime(entry.mod_time) : ""
      }}</span>
      <span class="entry-item__size">{{
        entry.size >= 0 ? $.formatBytes(entry.size) : ""
      }}</span>
    </span>
    <i-icon
      v-if="entry.meta.is_mount"
      class="entry-item__mount-icon"
      svg="#icon-path"
    />
  </div>
</template>
<script>
import { filenameExt } from '@/utils'

export default {
  name: 'EntryItem',
  props: {
    entry: {
      type: Object,
      required: true
    },
    icon: {
      type: String
    }
  },
  computed: {
    ext () {
      return filenameExt(this.entry.name)
    }
  }
}
</script>
<style lang="scss">
.entry-item {
  position: relative;
  display: flex;
  cursor: pointer;
  padding: 4px 16px;
}

.entry-item__icon {
  margin-right: 0.5em;
  font-size: 42px;
}

.entry-item__info {
  flex: 1;
  display: flex;
  align-items: center;
  overflow: hidden;
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
  width: 80px;
  font-size: 14px;
  text-align: right;
  white-space: nowrap;
}

.entry-item__mount-icon {
  position: absolute;
  bottom: 10px;
  left: 40px;
  color: #70a1ff;
}
</style>
