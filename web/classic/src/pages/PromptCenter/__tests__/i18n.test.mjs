import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import i18next from 'i18next';

const source = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../..',
);
const files = [
  'pages/PromptCenter/index.jsx',
  'pages/PromptCenter/TokenPicker.jsx',
  'pages/PromptCenter/PromptCatalog.jsx',
  'pages/PromptCenter/PromptTemplateEditor.jsx',
  'pages/PromptCenter/TokenPromptManager.jsx',
  'pages/PromptCenter/TokenPromptInstallationModal.jsx',
  'pages/PromptCenter/api.js',
  'pages/PromptCenter/SkillSummary.jsx',
  'pages/PromptCenter/BundleSkillsEditor.jsx',
  'pages/PromptCenter/SkillResourcesEditor.jsx',
  'pages/PromptCenter/CategoryManager.jsx',
  'pages/PromptCenter/ArchiveImport.jsx',
  'pages/PromptCenter/RepositoryImport.jsx',
  'pages/PromptCenter/RepositorySource.jsx',
  'pages/PromptCenter/RepositoryUpdate.jsx',
  'pages/PromptCenter/TemplateDelete.jsx',
  'pages/PromptCenter/SkillStore.jsx',
  'components/table/tokens/TokenPromptBadge.jsx',
  'components/playground/TokenSourceSelector.jsx',
  'components/playground/ChatArea.jsx',
  'components/playground/CustomInputRender.jsx',
  'hooks/playground/useDataLoader.js',
];
const keys = [
  ...new Set(
    files.flatMap((file) =>
      [
        ...fs
          .readFileSync(path.join(source, file), 'utf8')
          .matchAll(/\bt\(\s*'([^']+)'/g),
      ].map((match) => match[1]),
    ),
  ),
];
const validationSource = fs.readFileSync(
  path.join(source, 'pages/PromptCenter/validation.js'),
  'utf8',
);
for (const match of validationSource.matchAll(
  /(?:return\s*|\?\s*|:\s*)'([^']+)'/g,
)) {
  keys.push(match[1]);
}

test('八份语言资源覆盖提示词页面，保留动态名称与数量占位符', async () => {
  const resources = {};
  for (const locale of ['en', 'zh', 'zh-CN', 'zh-TW', 'fr', 'ru', 'ja', 'vi']) {
    const dictionary = JSON.parse(
      fs.readFileSync(
        path.join(source, 'i18n/locales', `${locale}.json`),
        'utf8',
      ),
    );
    resources[locale] = dictionary;
    for (const key of keys) {
      assert.ok(dictionary.translation[key], `${locale}: ${key}`);
      assert.deepEqual(
        dictionary.translation[key].match(/{{\w+}}/g)?.sort(),
        key.match(/{{\w+}}/g)?.sort(),
        `${locale}: ${key}`,
      );
      assert.ok(
        !/^\?+$/.test(dictionary.translation[key]),
        `${locale}: ${key}`,
      );
    }
  }
  const instance = i18next.createInstance();
  await instance.init({ lng: 'zh', resources });
  assert.equal(instance.t('Skills'), '技能');
  assert.equal(
    instance.t('{{name}}: {{count}} installed prompts, {{enabled}} enabled', {
      name: 'codexQ',
      count: 2,
      enabled: 1,
    }),
    'codexQ：已安装 2 个提示词，已启用 1 个',
  );
});
