import type { Directive, DirectiveHook } from 'vue'
import { T } from '@/i18n'

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
  import('@/utils/marked').then(
    ({ default: render }) => {
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
