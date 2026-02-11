// rollup.config.mjs
import typescript from '@rollup/plugin-typescript';
import resolve from '@rollup/plugin-node-resolve';
import terser from '@rollup/plugin-terser';
import dts from 'rollup-plugin-dts';
import pkg from './package.json' with { type: "json" };

const external = [
  ...Object.keys(pkg.dependencies || {}),
  ...Object.keys(pkg.peerDependencies || {})
];

const packageName = 'TypedIDB';

const isStdout = process.argv.includes('--file') === false && process.argv.includes('-o') === false;

// Shared plugins for all builds
const commonPlugins = [
  typescript({
    tsconfig: './tsconfig.json',
    declaration: false,
  }),
  resolve()
];

export default [
  // UMD bundle (minified)
  {
    input: 'src/index.ts',
    output: {
      name: packageName,
      file: pkg.unpkg,
      format: 'umd',
      sourcemap: isStdout ? 'inline' : true,
      exports: 'named',
    },
    plugins: [
      ...commonPlugins,
      terser(),
    ],
  },

  // ESM bundle
  {
    input: 'src/index.ts',
    external,
    output: {
      name: packageName,
      file: 'dist/bundle/index.esm.js',
      format: 'es',
      sourcemap: true,
    },
    plugins: commonPlugins,
  },
  // Types bundle
  {
    input: 'src/index.ts',
    output: {
      file: 'dist/bundle/index.d.ts',
      format: 'es',
    },
    plugins: [dts()],
  },
];