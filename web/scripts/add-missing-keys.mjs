import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const keys = [
  'Built-in plugins only. Marketplace, upload, deletion and sandbox are unavailable in this version.',
  'Disabling task plugins stops new submissions. Existing tasks continue with their pinned version.',
  'Not activated',
  'Select a task plugin',
]
const newKeys = {
  en: keys,
  zh: ['仅支持内置插件。本版本暂不支持市场、上传、删除和沙盒。', '关闭任务插件后将停止新任务提交，已有任务继续使用固定版本处理。', '未激活', '选择任务插件'],
  'zh-TW': ['僅支援內建外掛。本版本暫不支援市集、上傳、刪除和沙盒。', '關閉任務外掛後將停止新任務提交，既有任務繼續使用固定版本處理。', '未啟用版本', '選擇任務外掛'],
  fr: ['Seuls les plugins intégrés sont disponibles. Le marché, le téléversement, la suppression et le bac à sable ne sont pas disponibles dans cette version.', 'La désactivation bloque les nouvelles tâches. Les tâches existantes continuent avec leur version fixée.', 'Non activé', 'Sélectionner un plugin de tâches'],
  ja: ['組み込みプラグインのみ対応しています。このバージョンではマーケット、アップロード、削除、サンドボックスは利用できません。', 'タスクプラグインを無効にすると新規タスクの送信が停止します。既存のタスクは固定されたバージョンで処理を続行します。', '未有効化', 'タスクプラグインを選択'],
  ru: ['Доступны только встроенные плагины. Каталог, загрузка, удаление и песочница в этой версии недоступны.', 'Отключение блокирует новые задачи. Существующие задачи продолжают работать с закреплённой версией.', 'Не активирован', 'Выберите плагин задач'],
  vi: ['Chỉ hỗ trợ plugin tích hợp. Phiên bản này chưa hỗ trợ kho plugin, tải lên, xóa và môi trường thử nghiệm.', 'Tắt plugin tác vụ sẽ chặn tác vụ mới. Các tác vụ hiện có tiếp tục dùng phiên bản đã cố định.', 'Chưa kích hoạt', 'Chọn plugin tác vụ'],
}

for (const [locale, values] of Object.entries(newKeys)) {
  const file = path.join(LOCALES_DIR, `${locale}.json`)
  const json = JSON.parse(await fs.readFile(file, 'utf8'))
  for (const [index, key] of keys.entries()) json.translation[key] = values[index]
  json.translation = Object.fromEntries(Object.entries(json.translation).sort(([a], [b]) => a.localeCompare(b)))
  await fs.writeFile(file, JSON.stringify(json, null, 2) + '\n', 'utf8')
  console.log(`${locale}: ${keys.length} translations applied`)
}
