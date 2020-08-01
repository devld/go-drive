import IIcon from './IIcon.vue'
import EntryIcon from './EntryItem/EntryIcon.vue'
import EntryLink from './EntryLink.vue'
import EntryItem from './EntryItem'
import EntryList from './EntryList.vue'
import PathBar from './PathBar.vue'
import ErrorView from './ErrorView.vue'
import DialogView from './DialogView.vue'
import FloatButton from './FloatButton.vue'

const components = {
  IIcon,
  EntryIcon, EntryLink, EntryList, EntryItem,
  PathBar, ErrorView, DialogView, FloatButton
}

export default {
  install (Vue) {
    Object.keys(components).forEach(key => {
      Vue.component(key, components[key])
    })
  }
}
