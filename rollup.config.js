import json from '@rollup/plugin-json'
import { nodeResolve } from '@rollup/plugin-node-resolve'

export default [
  {
    input: 'public/js/index.js',
    output: {
      file: 'public/dist/index.js',
      format: 'iife',
      name: 'music',
      sourcemap: 'inline',
    },
    plugins: [nodeResolve(), json()],
  },
]
