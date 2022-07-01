import type { Directive } from 'vue'

const callback: IntersectionObserverCallback = (entries) => {
  entries.forEach((e) => {
    if (e.isIntersecting) {
      const img = e.target as HTMLImageElement
      observer.unobserve(img)
      loadImage(img)
    }
  })
}

const observer = new IntersectionObserver(callback, {
  rootMargin: '100px 0px',
  threshold: 0,
})

function loadImage(img: HTMLImageElement) {
  const src = img.getAttribute('data-src')
  if (!src) return
  img.src = src
  img.removeAttribute('data-src')
}

export default {
  mounted(el, { value }) {
    el.src =
      'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAAAXNSR0IArs4c6QAAAAtJREFUGFdjYAACAAAFAAGq1chRAAAAAElFTkSuQmCC'
    el.dataset.src = value
    if (value) {
      observer.observe(el)
    }
  },
  updated(el, { value }) {
    observer.unobserve(el)
    el.dataset.src = value
    if (value) {
      observer.observe(el)
    }
  },
  beforeUnmount(el) {
    observer.unobserve(el)
  },
} as Directive
