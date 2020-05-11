
export default function (el, binding) {
  Promise.all([
    import(/* webpackChunkName: "md" */ 'marked'),
    import(/* webpackChunkName: "md" */ '@/utils/highlight')
  ]).then(([Marked, hljs]) => {
    el.innerHTML = Marked.default(binding.value)
    for (const code of el.querySelectorAll('pre>code')) {
      hljs.default.highlightBlock(code)
    }
  }).catch(e => {
    console.log(e)
    el.innerHTML = '<p style="text-align: center;">An error occurred while rendering markdown</p>'
  })
}
