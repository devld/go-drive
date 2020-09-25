import IIcon from './IIcon.vue'
import EntryIcon from './EntryItem/EntryIcon.vue'
import EntryLink from './EntryLink.vue'
import EntryItem from './EntryItem'
import EntryList from './EntryList.vue'
import PathBar from './PathBar.vue'
import ErrorView from './ErrorView.vue'
import DialogView from './DialogView.vue'
import FloatButton from './FloatButton.vue'
import SimpleButton from './SimpleButton.vue'
import SimpleFormItem from './Form/FormItem.vue'
import SimpleForm from './Form'
import ProgressBar from './ProgressBar.vue'

const components = {
  IIcon, SimpleButton, SimpleForm, SimpleFormItem,
  EntryIcon, EntryLink, EntryList, EntryItem,
  PathBar, ErrorView, DialogView, FloatButton, ProgressBar
}

export default {
  install (Vue) {
    Object.keys(components).forEach(key => {
      Vue.component(key, components[key])
    })
  }
}
