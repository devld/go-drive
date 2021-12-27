module.exports = {
  root: true,
  env: {
    node: true,
  },
  extends: ['plugin:vue/essential', '@vue/standard'],
  parserOptions: {
    parser: 'babel-eslint',
  },
  rules: {
    'space-before-function-paren': 'off',
    'comma-dangle': 'off',
    'template-curly-spacing': 'off',
    indent: 'off',
    'object-property-newline': 'off',
    'no-console':
      process.env.NODE_ENV === 'production'
        ? ['warn', { allow: ['warn', 'error'] }]
        : 'off',
    'no-debugger': process.env.NODE_ENV === 'production' ? 'warn' : 'off',
  },
}
