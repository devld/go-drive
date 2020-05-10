import Marked from 'marked'

export default function (el, binding) {
  el.innerHTML = Marked(binding.value)
}
