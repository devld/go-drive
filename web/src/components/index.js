import EntryList from './EntryList.vue'
import EntryItem from './EntryItem'
import PathBar from './PathBar.vue'

const components = {
  EntryList, EntryItem, PathBar
}

export default {
  install (Vue) {
    Object.keys(components).forEach(key => {
      Vue.component(key, components[key])
    })
  }
}
