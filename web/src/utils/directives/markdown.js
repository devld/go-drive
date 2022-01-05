import { T } from '@/i18n'

let marked
async function getRender() {
  if (marked) return marked
  return Promise.all([
    import('marked'),
    import('@/utils/highlight'),
    import('dompurify'),
  ]).then(([marked_, { default: hljs }, { default: DOMPurify }]) => {
    marked_.setOptions({
      highlight: (code, language) => {
        const validLanguage = hljs.getLanguage(language)
          ? language
          : 'plaintext'
        return hljs.highlight(validLanguage, code).value
      },
    })
    marked = (s) => {
      return DOMPurify.sanitize(marked_.parse(s))
    }
    return marked
  })
}

const render = (el, binding) => {
  getRender().then(
    (render) => {
      el.innerHTML = render(binding.value)
    },
    (e) => {
      console.error('markdown render error: ', e)
      el.innerHTML = `<p style="text-align: center;">${T('md.error')}</p>`
    }
  )
}

export default {
  beforeMount: render,
  updated: render,
}
