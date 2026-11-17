# Issue #403–#405 i18n 集成记录

**任务范围**：为 Issue #404 倍率保护（9 键）和 Issue #405 顶部导航无障碍标签（2 键）添加七语翻译。

**执行日期**：2026-11-16

**备份**：7 个 locale 文件已备份至 `D:\Temp\i18n-backup-20261116`，SHA256 前缀（前 16 位）：

| locale | SHA256 前缀      |
| ------ | ---------------- |
| en     | F683E381955CA292 |
| fr     | B34DE4362C98A8AE |
| ja     | A61A06350B3879CB |
| ru     | AC844B40ED648DBB |
| vi     | 13905F9C73D67F2C |
| zh-TW  | 4C6FAD5EEA9924E8 |
| zh     | 13626D0DEE39F474 |

## 11 键清单

**#404 倍率保护（9 键，来源：`web/src/features/keys/.../ratio-protection` 组件 t() 调用）**

| 英文键                                                                   | zh                                         | zh-TW                                      | fr                                                                           | ja                                                                 | ru                                                                                   | vi                                                               |
| ------------------------------------------------------------------------ | ------------------------------------------ | ------------------------------------------ | ---------------------------------------------------------------------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------ | ---------------------------------------------------------------- |
| `Check ratio protection before saving.`                                  | 保存前请检查倍率保护。                     | 儲存前請檢查倍率保護。                     | Vérifiez la protection du ratio avant d'enregistrer.                         | 保存前に倍率保護を確認してください。                               | Проверьте защиту коэффициента перед сохранением.                                     | Kiểm tra bảo vệ tỷ lệ trước khi lưu.                             |
| `Clear ratio protection`                                                 | 清除倍率保护                               | 清除倍率保護                               | Effacer la protection du ratio                                               | 倍率保護をクリア                                                   | Очистить защиту коэффициента                                                         | Xóa bảo vệ tỷ lệ                                                 |
| `Enter a finite ratio of zero or greater.`                               | 请输入大于或等于零的有限倍率值。           | 請輸入大於或等於零的有限倍率值。           | Entrez un ratio fini, zéro ou plus.                                          | ゼロ以上の有限な倍率を入力してください。                           | Введите конечный коэффициент, равный нулю или более.                                 | Nhập tỷ lệ hữu hạn từ 0 trở lên.                                 |
| `No group selected. Inherit your user group.`                            | 未选择分组，将继承用户分组。               | 未選擇分組，將繼承使用者分組。             | Aucun groupe sélectionné. Héritage du groupe utilisateur.                    | グループ未選択。ユーザーグループを継承します。                     | Группа не выбрана. Используется группа пользователя.                                 | Chưa chọn nhóm. Kế thừa nhóm người dùng.                         |
| `Ratio limit for {{name}}`                                               | {{name}} 的倍率上限                        | {{name}} 的倍率上限                        | Limite de ratio pour {{name}}                                                | {{name}} の倍率上限                                                | Ограничение коэффициента для {{name}}                                                | Giới hạn tỷ lệ cho {{name}}                                      |
| `Ratio protection`                                                       | 倍率保护                                   | 倍率保護                                   | Protection du ratio                                                          | 倍率保護                                                           | Защита коэффициента                                                                  | Bảo vệ tỷ lệ                                                     |
| `Select explicit groups to configure ratio protection.`                  | 请选择明确的分组以配置倍率保护。           | 請選擇明確的分組以設定倍率保護。           | Sélectionnez des groupes explicites pour configurer la protection.           | 倍率保護を設定するグループを明示的に選択してください。             | Выберите явные группы для настройки защиты коэффициента.                             | Chọn nhóm cụ thể để cấu hình bảo vệ tỷ lệ.                       |
| `Set a maximum group ratio. Empty or zero removes protection.`           | 设置最大分组倍率，空值或零将移除保护。     | 設定最大分組倍率，空值或零將移除保護。     | Définir un ratio groupe max. Vide ou zéro supprime la protection.            | グループ倍率の上限を設定。空またはゼロで保護を解除。               | Задайте максимальный коэффициент группы. Пусто или ноль снимает защиту.              | Đặt tỷ lệ nhóm tối đa. Để trống hoặc bằng 0 để xóa bảo vệ.       |
| `Stored ratio protection is invalid. Clear it explicitly before saving.` | 已存储的倍率保护无效，请在保存前明确清除。 | 已儲存的倍率保護無效，請在儲存前明確清除。 | La protection de ratio stockée est invalide. Effacez-la avant d'enregistrer. | 保存された倍率保護が無効です。保存前に明示的にクリアしてください。 | Сохранённая защита коэффициента недействительна. Явно очистите её перед сохранением. | Bảo vệ tỷ lệ đã lưu không hợp lệ. Hãy xóa rõ ràng trước khi lưu. |

**#405 顶部导航无障碍标签（2 键，来源：`web/src/components/layout/components/top-nav.tsx:62,115`）**

| 英文键            | zh       | zh-TW    | fr                 | ja                     | ru             | vi              |
| ----------------- | -------- | -------- | ------------------ | ---------------------- | -------------- | --------------- |
| `Navigation`      | 导航     | 導覽     | Navigation         | ナビゲーション         | Навигация      | Điều hướng      |
| `Navigation menu` | 导航菜单 | 導覽選單 | Menu de navigation | ナビゲーションメニュー | Меню навигации | Menu điều hướng |

## Classic 新键检查

现有报告及本轮 grep 结果均未发现 Classic（`web/classic/`）新增 i18n 键，不制造 Classic 改动。#404 的 Classic 三个用例各自通过，不代表整批通过。

## 实际执行与验证记录

1. 临时脚本 `web/scripts/add-missing-keys.mjs`（事先确认不存在），按 `json.translation[key]` 写入 11 键×7 语言；首次误写顶层，已纠正。
2. `bun run i18n:sync`：退出码 0，missing 0、extras 0。
3. 术语返修（zh/zh-TW/ja：比例→倍率，レート→倍率，限制→上限，共 8 键×3 语言=24 值）后重跑 `bun run i18n:sync`：退出码 0。
4. 验证脚本 `node D:\Temp\verify-i18n-r3.mjs`：退出码 0；11 键×7 语言共 77/77 值存在，`{{name}}` 占位符保留，相对 `D:/Temp/i18n-backup-20261116-r2` 的 7770 个原键值不变，每语言 7781 键；105 个 MCP 条目仅通过「原键不变」检查覆盖。
5. 临时脚本与验证脚本均已删除；仅对本记录及两份索引文档运行 oxfmt。

## 未验边界

- Classic 本轮无新增 i18n 键或实现；既有三个用例各自通过，不称整批通过；真实后端与生产环境未验证
- 真实运行时渲染（无浏览器/生产验收）
- 全量 typecheck / build 未运行（禁止）
