const observer = new IntersectionObserver(callback, {
  rootMargin: '100px 0px',
  threshold: 0,
})

function callback(entries) {
  entries.forEach((e) => {
    if (e.isIntersecting) {
      const img = e.target
      observer.unobserve(img)
      loadImage(img)
    }
  })
}

function loadImage(img) {
  const src = img.getAttribute('data-src')
  if (!src) return
  img.src = src
  img.removeAttribute('data-src')
}

export default {
  mounted(el, { value }) {
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
}
