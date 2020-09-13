
import CopyMoveView from '@/views/HandlerViews/CopyMoveView.vue'

function createView (move) {
  return {
    name: 'CopyView',
    props: {
      entry: { type: [Object, Array], required: true },
      entries: { type: Array }
    },
    render (h) {
      return h(CopyMoveView, {
        props: { entry: this.entry, move },
        on: {
          update: () => { this.$emit('update') },
          close: () => { this.$emit('close') }
        }
      })
    }
  }
}

const CopyView = createView(false)
const MoveView = createView(true)

export const copy = {
  name: 'copy',
  display: {
    name: 'Copy to',
    description: 'Copy files',
    icon: '#icon-copy'
  },
  view: {
    name: 'CopyView',
    component: CopyView
  },
  multiple: true,
  supports: () => true
}

export const move = {
  name: 'move',
  display: {
    name: 'Move to',
    description: 'Move files',
    icon: '#icon-move'
  },
  view: {
    name: 'MoveView',
    component: MoveView
  },
  multiple: true,
  supports: (entry) => Array.isArray(entry) ? !entry.some(e => !e.meta.can_write) : entry.meta.can_write
}
