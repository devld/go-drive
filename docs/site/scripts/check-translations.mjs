#!/usr/bin/env node

import { createHash } from 'node:crypto'
import { readdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const writeMode = process.argv.includes('--write')
const unknownArgs = process.argv.slice(2).filter((arg) => arg !== '--write')
const excludedDirectories = new Set([
  '.git',
  '.github',
  '.vitepress',
  'node_modules',
  'public',
  'scripts',
])

if (unknownArgs.length > 0) {
  console.error(`Unknown argument(s): ${unknownArgs.join(', ')}`)
  console.error('Usage: node scripts/check-translations.mjs [--write]')
  process.exit(2)
}

async function walk(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    if (entry.isDirectory() && excludedDirectories.has(entry.name)) continue

    const absolutePath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...await walk(absolutePath))
    } else if (entry.isFile() && entry.name.endsWith('.md')) {
      files.push(absolutePath)
    }
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

  return { data, frontMatter: match[0] }
}

function normalize(contents) {
  return `${contents.replace(/\r\n/g, '\n').replace(/\n*$/, '')}\n`
}

function sourceHash(contents) {
  return createHash('sha256').update(normalize(contents), 'utf8').digest('hex')
}

function setSourceHash(contents, parsed, hash, relativePath) {
  if (!Object.hasOwn(parsed.data, 'source_hash')) {
    throw new Error(`${relativePath}: missing source_hash in front matter`)
  }

  const updatedFrontMatter = parsed.frontMatter.replace(
    /(^|\n)source_hash:\s*[^\r\n]*/,
    `$1source_hash: ${hash}`
  )
  return contents.replace(parsed.frontMatter, updatedFrontMatter)
}

const markdownFiles = await walk(root)
const pages = []
const errors = []

for (const absolutePath of markdownFiles) {
  const relativePath = path.relative(root, absolutePath).split(path.sep).join('/')
  const contents = await readFile(absolutePath, 'utf8')

  try {
    const parsed = parseFrontMatter(contents, relativePath)
    if (!parsed.data.lang) errors.push(`${relativePath}: missing lang`)
    if (!parsed.data.translation_key) errors.push(`${relativePath}: missing translation_key`)
    pages.push({ absolutePath, relativePath, contents, parsed })
  } catch (error) {
    errors.push(error.message)
  }
}

const supportedLanguages = new Set(['zh-CN', 'en'])
for (const page of pages) {
  if (page.parsed.data.lang && !supportedLanguages.has(page.parsed.data.lang)) {
    errors.push(`${page.relativePath}: unsupported lang ${page.parsed.data.lang}`)
  }
}

const pagesByKey = new Map()
for (const page of pages) {
  const key = page.parsed.data.translation_key
  if (!key) continue
  if (!pagesByKey.has(key)) pagesByKey.set(key, [])
  pagesByKey.get(key).push(page)
}

let updatedCount = 0
for (const [key, translations] of pagesByKey) {
  const sources = translations.filter((page) => page.parsed.data.lang === 'en')
  const zhPages = translations.filter((page) => page.parsed.data.lang === 'zh-CN')

  if (sources.length !== 1) {
    errors.push(`${key}: expected exactly one en page, found ${sources.length}`)
    continue
  }
  if (zhPages.length !== 1) {
    errors.push(`${key}: expected exactly one zh-CN page, found ${zhPages.length}`)
    continue
  }

  const source = sources[0]
  const translation = zhPages[0]
  const expectedTranslationPath = `zh-CN/${source.relativePath}`
  if (translation.relativePath !== expectedTranslationPath) {
    errors.push(`${key}: expected zh-CN page at ${expectedTranslationPath}, found ${translation.relativePath}`)
  }

  const expectedHash = sourceHash(source.contents)
  const recordedHash = translation.parsed.data.source_hash

  if (writeMode) {
    try {
      const updated = setSourceHash(translation.contents, translation.parsed, expectedHash, translation.relativePath)
      if (updated !== translation.contents) {
        await writeFile(translation.absolutePath, normalize(updated), 'utf8')
        updatedCount += 1
      }
    } catch (error) {
      errors.push(error.message)
    }
  } else if (recordedHash !== expectedHash) {
    errors.push(
      `${translation.relativePath}: stale translation for ${source.relativePath}; ` +
      `expected source_hash ${expectedHash}, found ${recordedHash || '(missing)'}`
    )
  }
}

for (const page of pages) {
  const key = page.parsed.data.translation_key
  if (!key) continue
  if (page.parsed.data.lang === 'zh-CN' && !page.relativePath.startsWith('zh-CN/')) {
    errors.push(`${page.relativePath}: zh-CN page must be under zh-CN/`)
  }
  if (page.parsed.data.lang === 'en' && page.relativePath.startsWith('zh-CN/')) {
    errors.push(`${page.relativePath}: English page must not be under zh-CN/`)
  }
}

if (errors.length > 0) {
  console.error(`Translation check failed with ${errors.length} error(s):`)
  for (const error of errors) console.error(`- ${error}`)
  process.exit(1)
}

const sourceCount = pages.filter((page) => page.parsed.data.lang === 'en').length
const translationCount = pages.filter((page) => page.parsed.data.lang === 'zh-CN').length
if (writeMode) {
  console.log(`Updated ${updatedCount} source hash(es); ${translationCount} translation pair(s) are complete.`)
} else {
  console.log(`Translation check passed: ${sourceCount} en and ${translationCount} zh-CN pages are paired and current.`)
}
