#!/usr/bin/env node

import { readdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const dist = path.join(root, '.vitepress', 'dist')
const siteUrl = 'https://go-drive.top'
const requiredMeta = [
  'og:type',
  'og:site_name',
  'og:title',
  'og:description',
  'og:url',
  'og:image',
  'og:image:width',
  'og:image:height',
  'og:image:alt',
  'og:locale',
  'twitter:card',
  'twitter:title',
  'twitter:description',
  'twitter:image',
  'twitter:image:alt'
]
const errors = []
const socialImageUrl = `${siteUrl}/social-card.png`

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    if (['.vitepress', 'node_modules', 'public', 'scripts'].includes(entry.name)) continue
    const absolutePath = path.join(directory, entry.name)
    if (entry.isDirectory()) files.push(...await walk(absolutePath))
    if (entry.isFile() && entry.name.endsWith('.md')) files.push(absolutePath)
  }
  return files
}

function parseFrontMatter(contents, relativePath) {
  const match = contents.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/)
  if (!match) throw new Error(`${relativePath}: missing YAML front matter`)

  const data = {}
  for (const line of match[1].split(/\r?\n/)) {
    const field = line.match(/^([A-Za-z_][A-Za-z0-9_-]*):\s*(.*?)\s*$/)
    if (!field) continue
    let value = field[2]
    if ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1)
    }
    data[field[1]] = value
  }
  return data
}

function pageUrl(relativePath) {
  const pathname = relativePath
    .replace(/(^|\/)index\.md$/, '$1')
    .replace(/\.md$/, '.html')
  return new URL(pathname, `${siteUrl}/`).href
}

function outputPath(relativePath) {
  return path.join(dist, relativePath.replace(/\.md$/, '.html'))
}

function attributes(tag) {
  return Object.fromEntries(
    [...tag.matchAll(/([:\w-]+)="([^"]*)"/g)].map((match) => [match[1], match[2]])
  )
}

function decodeHtml(value) {
  return value
    .replaceAll('&amp;', '&')
    .replaceAll('&quot;', '"')
    .replaceAll('&#39;', "'")
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
}

function metaContent(html, key) {
  for (const match of html.matchAll(/<meta\s+[^>]*>/g)) {
    const attrs = attributes(match[0])
    if (attrs.name === key || attrs.property === key) return decodeHtml(attrs.content || '')
  }
  return undefined
}

function linkHref(html, rel) {
  for (const match of html.matchAll(/<link\s+[^>]*>/g)) {
    const attrs = attributes(match[0])
    if (attrs.rel === rel) return decodeHtml(attrs.href || '')
  }
  return undefined
}

const markdownFiles = await walk(root)
const pages = []
for (const absolutePath of markdownFiles) {
  const relativePath = path.relative(root, absolutePath).split(path.sep).join('/')
  const contents = await readFile(absolutePath, 'utf8')
  const frontmatter = parseFrontMatter(contents, relativePath)
  pages.push({ relativePath, frontmatter })
}

const titlesByLanguage = new Map()
const descriptionsByLanguage = new Map()
const sitemap = await readFile(path.join(dist, 'sitemap.xml'), 'utf8')

for (const page of pages) {
  const { relativePath, frontmatter } = page
  const label = relativePath
  const lang = frontmatter.lang
  const canonical = pageUrl(relativePath)
  const html = await readFile(outputPath(relativePath), 'utf8')
  const expectedTitle = frontmatter.titleTemplate === 'false'
    ? frontmatter.title
    : `${frontmatter.title} | go-drive`

  if (!frontmatter.title) errors.push(`${label}: missing title`)
  if (!frontmatter.description) errors.push(`${label}: missing description`)
  if ((frontmatter.description || '').length < 35) {
    errors.push(`${label}: description must contain at least 35 characters`)
  }

  const titleMatch = html.match(/<title>([\s\S]*?)<\/title>/)
  const renderedTitle = titleMatch ? decodeHtml(titleMatch[1]) : undefined
  if (renderedTitle !== expectedTitle) {
    errors.push(`${label}: expected title "${expectedTitle}", found "${renderedTitle || '(missing)'}"`)
  }
  if (metaContent(html, 'description') !== frontmatter.description) {
    errors.push(`${label}: rendered description does not match front matter`)
  }
  if (linkHref(html, 'canonical') !== canonical) {
    errors.push(`${label}: expected canonical ${canonical}`)
  }

  for (const key of requiredMeta) {
    if (!metaContent(html, key)) errors.push(`${label}: missing ${key}`)
  }
  if (metaContent(html, 'og:url') !== canonical) errors.push(`${label}: og:url must match canonical`)
  if (metaContent(html, 'og:image') !== socialImageUrl) errors.push(`${label}: unexpected og:image`)
  if (metaContent(html, 'twitter:image') !== socialImageUrl) errors.push(`${label}: unexpected twitter:image`)
  if (metaContent(html, 'twitter:card') !== 'summary_large_image') {
    errors.push(`${label}: twitter:card must use summary_large_image`)
  }
  if (metaContent(html, 'og:description') !== frontmatter.description) {
    errors.push(`${label}: og:description must match description`)
  }
  if (metaContent(html, 'twitter:description') !== frontmatter.description) {
    errors.push(`${label}: twitter:description must match description`)
  }

  const h1Count = [...html.matchAll(/<h1(?:\s|>)/g)].length
  if (h1Count !== 1) errors.push(`${label}: expected one h1, found ${h1Count}`)

  const jsonLdMatch = html.match(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/)
  if (!jsonLdMatch) {
    errors.push(`${label}: missing JSON-LD`)
  } else {
    try {
      const jsonLd = JSON.parse(jsonLdMatch[1])
      const isHome = relativePath === 'index.md' || relativePath === 'zh-CN/index.md'
      const types = (Array.isArray(jsonLd) ? jsonLd : [jsonLd]).map((item) => item['@type'])
      const expectedTypes = isHome ? ['WebSite'] : ['TechArticle', 'BreadcrumbList']
      for (const type of expectedTypes) {
        if (!types.includes(type)) errors.push(`${label}: JSON-LD missing ${type}`)
      }
      const article = (Array.isArray(jsonLd) ? jsonLd : []).find((item) => item['@type'] === 'TechArticle')
      if (article && (article.url !== canonical || article.description !== frontmatter.description)) {
        errors.push(`${label}: TechArticle URL or description does not match page metadata`)
      }
      const breadcrumb = (Array.isArray(jsonLd) ? jsonLd : []).find((item) => item['@type'] === 'BreadcrumbList')
      if (!isHome && breadcrumb?.itemListElement?.at(-1)?.item !== canonical) {
        errors.push(`${label}: breadcrumb must end at canonical URL`)
      }
    } catch (error) {
      errors.push(`${label}: invalid JSON-LD: ${error.message}`)
    }
  }

  if (!sitemap.includes(`<loc>${canonical}</loc>`)) errors.push(`${label}: canonical missing from sitemap`)
  if (!sitemap.includes(`hreflang="${lang}" href="${canonical}"`)) {
    errors.push(`${label}: language alternate missing from sitemap`)
  }

  const titleKey = `${lang}\0${expectedTitle}`
  if (titlesByLanguage.has(titleKey)) {
    errors.push(`${label}: duplicate title also used by ${titlesByLanguage.get(titleKey)}`)
  } else {
    titlesByLanguage.set(titleKey, label)
  }
  const descriptionKey = `${lang}\0${frontmatter.description}`
  if (descriptionsByLanguage.has(descriptionKey)) {
    errors.push(`${label}: duplicate description also used by ${descriptionsByLanguage.get(descriptionKey)}`)
  } else {
    descriptionsByLanguage.set(descriptionKey, label)
  }
}

const sitemapPageCount = [...sitemap.matchAll(/<url>/g)].length
if (sitemapPageCount !== pages.length) {
  errors.push(`sitemap.xml: expected ${pages.length} URLs, found ${sitemapPageCount}`)
}

const robots = await readFile(path.join(dist, 'robots.txt'), 'utf8')
if (!/^User-agent: \*$/m.test(robots) || !/^Allow: \/$/m.test(robots)) {
  errors.push('robots.txt: must allow all crawlers')
}
if (!robots.includes(`Sitemap: ${siteUrl}/sitemap.xml`)) {
  errors.push('robots.txt: missing sitemap URL')
}

const socialImage = await readFile(path.join(dist, 'social-card.png'))
const imageWidth = socialImage.readUInt32BE(16)
const imageHeight = socialImage.readUInt32BE(20)
if (imageWidth < 1200 || imageHeight < 630) {
  errors.push(`social-card.png: expected at least 1200x630, found ${imageWidth}x${imageHeight}`)
}

if (errors.length > 0) {
  console.error(`SEO check failed with ${errors.length} error(s):`)
  for (const error of errors) console.error(`- ${error}`)
  process.exit(1)
}

console.log(`SEO check passed: ${pages.length} pages have unique metadata, canonical URLs, social tags, JSON-LD, and sitemap entries.`)
