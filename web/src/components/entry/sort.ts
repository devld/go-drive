import { Entry } from '@/types'

export const SORTS_METHOD: O<(a: Entry, b: Entry) => number> = {
  name_asc: (a, b) =>
    a.type.localeCompare(b.type) || a.name.localeCompare(b.name),
  name_desc: (a, b) =>
    -a.type.localeCompare(b.type) || -a.name.localeCompare(b.name),
  mod_time_asc: (a, b) =>
    a.type.localeCompare(b.type) ||
    a.modTime - b.modTime ||
    a.name.localeCompare(b.name),
  mod_time_desc: (a, b) =>
    -a.type.localeCompare(b.type) ||
    b.modTime - a.modTime ||
    a.name.localeCompare(b.name),
  size_asc: (a, b) =>
    a.type.localeCompare(b.type) ||
    a.size - b.size ||
    a.name.localeCompare(b.name),
  size_desc: (a, b) =>
    -a.type.localeCompare(b.type) ||
    b.size - a.size ||
    a.name.localeCompare(b.name),
}

export const sortModes = Object.keys(SORTS_METHOD).map((key) => ({
  key,
  name: `app.sort.${key}`,
}))
