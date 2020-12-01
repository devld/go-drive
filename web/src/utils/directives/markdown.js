import { T } from '@/i18n'

let marked
async function getRender () {
  if (marked) return marked
  return Promise.all([
    import(/* webpackChunkName: "md" */ 'marked'),
    import(/* webpackChunkName: "md" */ '@/utils/highlight'),
    import(/* webpackChunkName: "md" */ 'dompurify')
  ]).then(([{ default: marked_ }, { default: hljs }, { default: DOMPurify }]) => {
    marked_.setOptions({
      highlight: (code, language) => {
        const validLanguage = hljs.getLanguage(language) ? language : 'plaintext'
        return hljs.highlight(validLanguage, code).value
      }
    })
    marked = (s) => {
      return DOMPurify.sanitize(marked_(s))
    }
    return marked
  })
}

export default function (el, binding) {
  getRender().then(render => {
    el.innerHTML = render(binding.value)
  }, e => {
    el.innerHTML = `<p style="text-align: center;">${T('md.error')}</p>`
  })
}
