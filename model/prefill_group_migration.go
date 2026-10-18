package model

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const prefillGroupNameIndex = "uk_prefill_name"

type conflictingPrefillGroupUniqueness struct {
	schemaName  string
	tableName   string
	constraints []string
	indexes     []string
}

type prefillGroupNameIndexState struct {
	exists bool
	valid  bool
}

func (conflicts conflictingPrefillGroupUniqueness) empty() bool {
	return len(conflicts.constraints) == 0 && len(conflicts.indexes) == 0
}

func inspectConflictingPrefillGroupUniqueness(db *gorm.DB, tableName string) (conflictingPrefillGroupUniqueness, error) {
	var conflicts conflictingPrefillGroupUniqueness
	var relation struct {
		SchemaName string
		TableName  string
	}
	if err := db.Raw(`
SELECT namespace_meta.nspname AS schema_name, table_meta.relname AS table_name
FROM pg_catalog.pg_class AS table_meta
JOIN pg_catalog.pg_namespace AS namespace_meta ON namespace_meta.oid = table_meta.relnamespace
WHERE table_meta.oid = to_regclass(?)`, tableName).Scan(&relation).Error; err != nil {
		return conflicts, fmt.Errorf("inspect prefill group relation: %w", err)
	}
	if relation.SchemaName == "" {
		return conflicts, nil
	}
	conflicts.schemaName, conflicts.tableName = relation.SchemaName, relation.TableName
	if err := db.Raw(`
SELECT constraint_meta.conname
FROM pg_catalog.pg_constraint AS constraint_meta
WHERE constraint_meta.conrelid = to_regclass(?)
  AND constraint_meta.contype = 'u'
  AND cardinality(constraint_meta.conkey) = 1
  AND EXISTS (
      SELECT 1
      FROM pg_catalog.pg_attribute AS attribute_meta
      WHERE attribute_meta.attrelid = constraint_meta.conrelid
        AND attribute_meta.attnum = constraint_meta.conkey[1]
        AND attribute_meta.attname = ?
  )
ORDER BY constraint_meta.conname`, tableName, "name").Scan(&conflicts.constraints).Error; err != nil {
		return conflicts, fmt.Errorf("inspect conflicting prefill group unique constraints: %w", err)
	}

	if err := db.Raw(`
SELECT index_class.relname
FROM pg_catalog.pg_index AS index_meta
JOIN pg_catalog.pg_class AS index_class
  ON index_class.oid = index_meta.indexrelid
JOIN pg_catalog.pg_attribute AS attribute_meta
  ON attribute_meta.attrelid = index_meta.indrelid
 AND attribute_meta.attnum = index_meta.indkey[0]
WHERE index_meta.indrelid = to_regclass(?)
  AND index_meta.indisunique
  AND NOT index_meta.indisprimary
  AND index_meta.indpred IS NULL
  AND index_meta.indexprs IS NULL
  AND index_meta.indnatts = 1
  AND attribute_meta.attname = ?
  AND NOT EXISTS (
      SELECT 1
      FROM pg_catalog.pg_constraint AS constraint_meta
      WHERE constraint_meta.conindid = index_meta.indexrelid
  )
ORDER BY index_class.relname`, tableName, "name").Scan(&conflicts.indexes).Error; err != nil {
		return conflicts, fmt.Errorf("inspect conflicting prefill group unique indexes: %w", err)
	}

	return conflicts, nil
}

func inspectPrefillGroupNameIndex(db *gorm.DB, tableName string) (prefillGroupNameIndexState, error) {
	var state struct {
		Exists bool `gorm:"column:index_exists"`
		Valid  bool `gorm:"column:index_valid"`
	}
	if err := db.Raw(`
SELECT count(*) > 0 AS index_exists,
       COALESCE(bool_or(
           index_meta.indisunique
           AND index_meta.indisvalid
           AND index_meta.indisready
           AND NOT index_meta.indisprimary
           AND index_meta.indexprs IS NULL
           AND index_meta.indnatts = 1
           AND attribute_meta.attname = ?
           AND pg_get_expr(index_meta.indpred, index_meta.indrelid) = '(deleted_at IS NULL)'
       ), false) AS index_valid
FROM pg_catalog.pg_index AS index_meta
JOIN pg_catalog.pg_class AS index_class
  ON index_class.oid = index_meta.indexrelid
LEFT JOIN pg_catalog.pg_attribute AS attribute_meta
  ON attribute_meta.attrelid = index_meta.indrelid
 AND attribute_meta.attnum = index_meta.indkey[0]
WHERE index_meta.indrelid = to_regclass(?)
  AND index_class.relname = ?`, "name", tableName, prefillGroupNameIndex).Scan(&state).Error; err != nil {
		return prefillGroupNameIndexState{}, fmt.Errorf("inspect prefill group partial unique index: %w", err)
	}
	return prefillGroupNameIndexState{exists: state.Exists, valid: state.Valid}, nil
}

// dropConflictingPrefillGroupUniqueness 在调用方持有排他锁的事务中删除已确认的旧对象。
func dropConflictingPrefillGroupUniqueness(tx *gorm.DB, conflicts conflictingPrefillGroupUniqueness) error {
	if conflicts.schemaName == "" || conflicts.tableName == "" {
		return fmt.Errorf("prefill group relation is not resolved")
	}
	// 老驱动会把对象名称中的点拆成 schema；目录名必须逐段双引号转义，
	// 并固定目标表的实际 schema，不能让 search_path 选中其他表的同名索引。
	schemaPrefix := `"` + strings.ReplaceAll(conflicts.schemaName, `"`, `""`) + `".`
	table := clause.Table{Name: schemaPrefix + `"` + strings.ReplaceAll(conflicts.tableName, `"`, `""`) + `"`, Raw: true}
	for _, constraintName := range conflicts.constraints {
		constraint := clause.Column{Name: `"` + strings.ReplaceAll(constraintName, `"`, `""`) + `"`, Raw: true}
		if err := tx.Exec("ALTER TABLE ? DROP CONSTRAINT ?", table, constraint).Error; err != nil {
			return fmt.Errorf("drop conflicting prefill group constraint %q: %w", constraintName, err)
		}
	}
	for _, indexName := range conflicts.indexes {
		index := clause.Column{Name: schemaPrefix + `"` + strings.ReplaceAll(indexName, `"`, `""`) + `"`, Raw: true}
		if err := tx.Exec("DROP INDEX ?", index).Error; err != nil {
			return fmt.Errorf("drop conflicting prefill group index %q: %w", indexName, err)
		}
	}
	return nil
}

// migratePrefillGroupUniqueness 在 AutoMigrate 前替换 PostgreSQL 的全局 name
// 唯一约束；按定义识别导入或改名后的索引，保留复合、表达式和部分索引语义。
func migratePrefillGroupUniqueness(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("migrate prefill group uniqueness: database is nil")
	}
	if db.Dialector.Name() != "postgres" {
		return nil
	}

	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(&PrefillGroup{}); err != nil {
		return fmt.Errorf("parse prefill group schema: %w", err)
	}
	tableName := statement.Schema.Table
	conflicts, err := inspectConflictingPrefillGroupUniqueness(db, tableName)
	if err != nil {
		return err
	}
	if conflicts.empty() {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		migrator := tx.Migrator()
		if !migrator.HasTable(&PrefillGroup{}) {
			return nil
		}

		if err := tx.Exec(
			"LOCK TABLE ? IN ACCESS EXCLUSIVE MODE",
			clause.Table{Name: tableName},
		).Error; err != nil {
			return fmt.Errorf("lock prefill groups for uniqueness migration: %w", err)
		}

		conflicts, err := inspectConflictingPrefillGroupUniqueness(tx, tableName)
		if err != nil {
			return err
		}
		if conflicts.empty() {
			return nil
		}
		if !migrator.HasColumn(&PrefillGroup{}, "DeletedAt") {
			if err := migrator.AddColumn(&PrefillGroup{}, "DeletedAt"); err != nil {
				return fmt.Errorf("add prefill groups deleted_at column: %w", err)
			}
		}

		// 全局约束可能占用目标名称；在排他锁内替换，任一步失败恢复旧约束。
		if err := dropConflictingPrefillGroupUniqueness(tx, conflicts); err != nil {
			return err
		}

		targetIndex, err := inspectPrefillGroupNameIndex(tx, tableName)
		if err != nil {
			return err
		}
		if !targetIndex.exists {
			if err := migrator.CreateIndex(&PrefillGroup{}, prefillGroupNameIndex); err != nil {
				return fmt.Errorf("create prefill group partial unique index: %w", err)
			}
			targetIndex, err = inspectPrefillGroupNameIndex(tx, tableName)
			if err != nil {
				return err
			}
		}
		if !targetIndex.valid {
			return fmt.Errorf("prefill group index %q has an unexpected definition", prefillGroupNameIndex)
		}

		return nil
	})
}
