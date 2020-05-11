import EntryItem from './EntryItem'
import EntryList from './EntryList.vue'
import PathBar from './PathBar.vue'
import ErrorView from './ErrorView.vue'

const components = {
  EntryList, EntryItem, PathBar, ErrorView
}

export default {
  install (Vue) {
    Object.keys(components).forEach(key => {
      Vue.component(key, components[key])
    })
  }
}
