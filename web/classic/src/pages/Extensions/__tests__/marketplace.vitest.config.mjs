import path from 'node:path';
import fs from 'node:fs';
import { fileURLToPath } from 'node:url';

// 可选测试依赖目录用于 Windows 工作树，避免 Classic React 18 混用 Default React 19。
const root = fileURLToPath(new URL('../../../../', import.meta.url));
const dependencyRoot = process.env.CLASSIC_TEST_DEPENDENCIES;
const aliases = [];
if (dependencyRoot) {
  const packages = JSON.parse(
    fs.readFileSync(path.join(root, 'package.json'), 'utf8'),
  );
  for (const name of [
    ...Object.keys(packages.dependencies),
    ...Object.keys(packages.devDependencies),
  ]) {
    if (fs.existsSync(path.join(dependencyRoot, name)))
      aliases.push({
        find: name,
        replacement: path.join(dependencyRoot, name).replaceAll('\\', '/'),
      });
  }
}
export default {
  root,
  resolve: { alias: aliases },
  esbuild: { jsx: 'automatic' },
  test: {
    environment: 'jsdom',
    include: ['src/pages/Extensions/__tests__/marketplace.compat.test.jsx'],
    setupFiles: ['./scripts/setup-compat-tests.mjs'],
    pool: 'forks',
    poolOptions: { forks: { singleFork: true } },
    testTimeout: 15000,
  },
};
