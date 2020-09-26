<template>
  <div class="copy-move-view">
    <entry-list-view
      :path="path"
      :filter="isDir"
      @entry-click="entryClicked"
      @path-change="pathChanged"
      @entries-load="entriesLoad"
    />

    <div class="footer">
      <span class="copy-move-override">
        <input type="checkbox" v-model="override" />Override?
      </span>
      <simple-button type="info" @click="$emit('close')">Cancel</simple-button>
      <simple-button :disabled="!currentDirCanWrite" @click="doCopyOrMove">{{
        move ? "Move to here" : "Copy to here"
      }}</simple-button>
    </div>
  </div>
</template>
<script>
import EntryListView from '@/views/EntryListView'
import { copyEntry, deleteTask, getEntry, moveEntry } from '@/api'
import { pathClean, pathJoin, taskDone, TASK_CANCELLED } from '@/utils'

export default {
  name: 'CopyMoveView',
  components: { EntryListView },
  props: {
    move: {
      type: Boolean,
      required: true
    },
    entry: {
      type: [Object, Array],
      required: true
    }
  },
  data () {
    return {
      path: '',
      currentDir: null,
      override: false
    }
  },
  computed: {
    currentDirCanWrite () {
      return this.currentDir && this.currentDir.meta.can_write
    }
  },
  methods: {
    async doCopyOrMove () {
      let canceled = false
      const entries = Array.isArray(this.entry) ? [...this.entry] : [this.entry]
      let task
      const onCancel = () => {
        canceled = true
        return task && deleteTask(task.id)
      }
      try {
        for (const i in entries) {
          if (canceled) break
          const entry = entries[i]
          const dest = pathClean(pathJoin(this.path, entry.name))
          this.$loading({ text: `${this.move ? 'Moving' : 'Copying'} ${entry.name}`, onCancel })
          const copyOrMove = this.move ? moveEntry : copyEntry
          await taskDone(
            copyOrMove(entry.path, dest, this.override),
            t => {
              if (canceled) return false
              task = t
              this.$loading({
                text: `${this.move ? 'Moving' : 'Copying'} ${entry.name} ` +
                  `${task.progress.loaded}/${task.progress.total}`,
                onCancel
              })
            }
          )
        }
      } catch (e) {
        if (e === TASK_CANCELLED) return
        this.$alert(e.message)
      } finally {
        this.$emit('update')
        this.$emit('close')
        this.$loading()
      }
    },
    entryClicked ({ entry, event }) {
      event.preventDefault()
      this.path = entry.path
      this.currentDir = null
    },
    pathChanged ({ path, event }) {
      event.preventDefault()
      this.path = path
      this.currentDir = null
    },
    entriesLoad () {
      getEntry(this.path).then(entry => { this.currentDir = entry }, () => { })
    },
    isDir (entry) {
      return entry.type === 'dir'
    }
  }
}
</script>
<style lang="scss">
.copy-move-view {
  position: relative;
  background-color: #fff;
  width: 320px;
  height: 50vh;
  padding-bottom: 60px;

  .entry-list-view {
    position: relative;
    height: 100%;
    overflow-x: hidden;
    overflow-y: auto;

    .path-bar {
      padding-top: 16px;
      padding-bottom: 10px;
      margin-bottom: 0;
      position: sticky;
      top: 0;
      background-color: #fff;
      box-shadow: 0 0 6px rgba(0, 0, 0, 0.1);
    }
  }

  .footer {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    padding: 16px;
    text-align: right;
    background-color: #fff;
    box-shadow: 0 0 6px rgba(0, 0, 0, 0.1);
  }

  .entry-item__modified-time {
    display: none;
  }

  .entry-item__size {
    display: none;
  }
}
</style>
