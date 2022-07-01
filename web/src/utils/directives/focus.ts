import type { Directive } from 'vue'

export default {
  mounted(el) {
    el.focus()
  },
} as Directive
