import EntryIcon from './EntryItem/EntryIcon.vue'
import EntryItem from './EntryItem'
import EntryList from './EntryList.vue'
import PathBar from './PathBar.vue'
import ErrorView from './ErrorView.vue'

const components = {
  EntryIcon, EntryList, EntryItem, PathBar, ErrorView
}

export default {
  install (Vue) {
    Object.keys(components).forEach(key => {
      Vue.component(key, components[key])
    })
  }
}
