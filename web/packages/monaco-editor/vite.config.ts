import { defineConfig } from 'vite'

const languageGroups: Record<string, string[]> = {
  web: ['html', 'handlebars', 'pug', 'razor', 'twig', 'liquid'],
  styles: ['css', 'scss', 'less'],
  config: [
    'xml',
    'yaml',
    'ini',
    'dockerfile',
    'hcl',
    'protobuf',
    'graphql',
    'bicep',
    'csp',
  ],
  shell: ['shell', 'powershell', 'bat'],
  markup: ['markdown', 'mdx', 'restructuredtext'],
  data: ['sql', 'mysql', 'pgsql', 'redis', 'cypher', 'sparql', 'msdax', 'kusto'],
  systems: [
    'cpp',
    'csharp',
    'java',
    'kotlin',
    'go',
    'rust',
    'swift',
    'objective-c',
    'scala',
  ],
}

// https://vitejs.dev/config/
export default defineConfig({
  base: './',
  server: {
    port: 9804,
  },
  build: {
    outDir: '../../public/code-editor',
    emptyOutDir: true,
    rolldownOptions: {
      output: {
        codeSplitting: {
          includeDependenciesRecursively: false,
          groups: [
            {
              debugName: 'language-grammars',
              maxSize: 80_000,
              name: (id) => {
                // Keep registrations in the entry and group only lazy grammar modules.
                const language = id.match(
                  /\/basic-languages\/([^/]+)\/(?![^/]*contribution)[^/]+\.js$/
                )?.[1]
                if (!language || ['javascript', 'typescript'].includes(language)) {
                  return
                }
                const group = Object.entries(languageGroups).find(
                  ([, languages]) => languages.includes(language)
                )?.[0]
                return `languages-${group ?? 'other'}`
              },
            },
          ],
        },
      },
    },
  },
})
