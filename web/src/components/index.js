import EntryIcon from './EntryItem/EntryIcon.vue'
import EntryItem from './EntryItem'
import EntryList from './EntryList.vue'
import PathBar from './PathBar.vue'
import ErrorView from './ErrorView.vue'
import DialogView from './DialogView.vue'

const components = {
  EntryIcon, EntryList, EntryItem, PathBar, ErrorView, DialogView
}

export default {
  install (Vue) {
    Object.keys(components).forEach(key => {
      Vue.component(key, components[key])
    })
  }
}
