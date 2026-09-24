import { defineConfig, mergeConfig } from 'vitest/config';
import { fileURLToPath } from 'node:url';
import baseConfig from '../../../../../vitest.config.mjs';

// 真实任务列会加载完整图标库，在本模块测试中预打包以避免依赖收集超时。
export default mergeConfig(
  baseConfig,
  defineConfig({
    resolve: {
      alias: {
        debug: fileURLToPath(
          new URL(
            '../../../../../node_modules/debug/src/browser.js',
            import.meta.url,
          ),
        ),
      },
    },
    test: {
      deps: {
        optimizer: {
          web: {
            enabled: true,
            include: ['@lobehub/icons'],
            esbuildOptions: {
              platform: 'node',
              banner: {
                js: "import { createRequire } from 'node:module'; const require = createRequire(import.meta.url);",
              },
            },
          },
        },
      },
    },
  }),
);
