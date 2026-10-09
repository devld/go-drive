import js from '@eslint/js'
import globals from 'globals'
import pluginVue from 'eslint-plugin-vue'
import { withVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import prettier from 'eslint-config-prettier/flat'

export default withVueTs(
  { rootDir: import.meta.dirname },
  {
    // Exclude release assets and preserve the legacy dotfile exclusion.
    ignores: ['**/dist/**', 'public/**', '**/.*'],
  },
  js.configs.recommended,
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
  prettier,
  {
    linterOptions: { reportUnusedDisableDirectives: 'off' },
    languageOptions: {
      ecmaVersion: 2021,
      globals: { ...globals.browser, ...globals.node, ...globals.es2021 },
    },
    rules: {
      'vue/require-default-prop': 'off',
      'vue/multi-word-component-names': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      // Keep the previous recommended rules rather than adopting new checks.
      'no-constant-binary-expression': 'off',
      'no-empty-static-block': 'off',
      'no-unassigned-vars': 'off',
      'no-unused-private-class-members': 'off',
      'no-useless-assignment': 'off',
      'preserve-caught-error': 'off',
      'no-constant-condition': ['error', { checkLoops: true }],
      'no-inner-declarations': 'error',
      'no-class-assign': 'error',
      'no-with': 'error',
      'vue/no-deprecated-delete-set': 'off',
      'vue/no-deprecated-model-definition': 'off',
      'vue/valid-define-options': 'off',
      'vue/no-required-prop-with-default': 'off',
      // These replace component-tags-order and ban-types in the new plugins.
      'vue/block-order': 'warn',
      '@typescript-eslint/no-empty-object-type': [
        'error',
        { allowInterfaces: 'always' },
      ],
      '@typescript-eslint/no-require-imports': [
        'error',
        { allowAsImport: true },
      ],
      '@typescript-eslint/no-unused-expressions': 'off',
      '@typescript-eslint/prefer-namespace-keyword': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { caughtErrors: 'none' }],
      '@typescript-eslint/ban-ts-comment': ['error', { 'ts-check': false }],
    },
  },
  {
    files: ['**/*.vue'],
    rules: {
      'no-undef': 'off',
      'vue/block-lang': 'off',
      // The old Vue config kept these core checks enabled in SFC scripts.
      'constructor-super': 'error',
      'getter-return': 'error',
      'no-const-assign': 'error',
      'no-dupe-args': 'error',
      'no-dupe-class-members': 'error',
      'no-dupe-keys': 'error',
      'no-func-assign': 'error',
      'no-import-assign': 'error',
      'no-obj-calls': 'error',
      'no-redeclare': 'error',
      'no-setter-return': 'error',
      'no-this-before-super': 'error',
      'no-unreachable': 'error',
      'no-unsafe-negation': 'error',
      'no-new-symbol': 'error',
    },
  }
)
