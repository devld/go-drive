import type { Directive, DirectiveHook } from 'vue'
import { T } from '@/i18n'

let marked: (s: string) => string
async function getRender() {
  if (marked) return marked
  return Promise.all([
    import('marked'),
    import('@/utils/highlight'),
    import('dompurify'),
  ]).then(
    ([{ marked: marked_ }, { default: hljs }, { default: DOMPurify }]) => {
      marked_.setOptions({
        highlight: (code, language) => {
          const validLanguage: string = hljs.getLanguage(language)
            ? language
            : 'plaintext'
          return hljs.highlight(code, { language: validLanguage }).value
        },
      })

      DOMPurify.addHook('afterSanitizeAttributes', (node) => {
        if ('target' in node) {
          node.setAttribute('target', '_blank')
        }
        if ('href' in node) {
          const a = node as HTMLAnchorElement
          const href = a.getAttribute('href') ?? ''
          if (href.startsWith('#')) {
            a.dataset.href = href
            a.href = 'javascript:;'
          }
        }
      })

      marked = (s) => {
        return DOMPurify.sanitize(marked_.parse(s))
      }
      return marked
    }
  )
}

const onAnchorClicked = function (this: HTMLAnchorElement, e: MouseEvent) {
  e.preventDefault()
  const href = this.dataset.href
  if (!href) return
  if (href.startsWith('#')) {
    const id = decodeURIComponent(href.substring(1))
    const targetEl = document.getElementById(id)
    if (targetEl) {
      targetEl.scrollIntoView({ behavior: 'smooth' })
    }
  } else {
    const a = document.createElement('a')
    a.target = '_blank'
    a.rel = 'nofollow noopener noreferrer'
    a.href = href
    a.click()
  }
}

const render: DirectiveHook = (el, binding) => {
  el._currentMarkdownContent = binding.value
  getRender().then(
    (render) => {
      if (el._currentMarkdownContent === el._renderedMarkdownContent) return
      el.innerHTML = render(el._currentMarkdownContent)
      el._renderedMarkdownContent = el._currentMarkdownContent

      const anchors = el.querySelectorAll('a[data-href]')
      anchors.forEach((a: HTMLAnchorElement) => {
        a.addEventListener('click', onAnchorClicked)
      })
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
} as Directive
