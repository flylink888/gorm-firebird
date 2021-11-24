package firebird

import (
	"database/sql"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
	"strings"
)

type Migrator struct {
	migrator.Migrator
	Dialector
}

type Column struct {
	name              string
	nullable          sql.NullString
	datatype          string
	maxLen            sql.NullInt64
	precision         sql.NullInt64
	scale             sql.NullInt64
	datetimePrecision sql.NullInt64
}

func (c Column) Name() string {
	//从数据库中查到的字段名,有空格需要去除
	return strings.TrimSpace(c.name)
}

func (c Column) DatabaseTypeName() string {
	return c.datatype
}

func (c Column) Length() (int64, bool) {
	if c.maxLen.Valid {
		return c.maxLen.Int64, c.maxLen.Valid
	}

	return 0, false
}

func (c Column) Nullable() (bool, bool) {
	if c.nullable.Valid {
		return c.nullable.String != "1", true
	}

	return false, false
}

// DecimalSize return precision int64, scale int64, ok bool
func (c Column) DecimalSize() (int64, int64, bool) {
	if c.precision.Valid {
		if c.scale.Valid {
			return c.precision.Int64, c.scale.Int64, true
		}

		return c.precision.Int64, 0, true
	}

	if c.datetimePrecision.Valid {
		return c.datetimePrecision.Int64, 0, true
	}

	return 0, 0, false
}

func (m Migrator) AlterColumn(value interface{}, field string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return m.DB.Exec(
				"ALTER TABLE ? ALTER COLUMN ? TYPE ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: field.DBName}, m.FullDataTypeOf(field),
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (m Migrator) RenameColumn(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {

		var field *schema.Field
		if f := stmt.Schema.LookUpField(oldName); f != nil {
			oldName = f.DBName
			field = f
		}

		if f := stmt.Schema.LookUpField(newName); f != nil {
			newName = f.DBName
			field = f
		}

		if field != nil {
			return m.DB.Exec(
				"ALTER TABLE ? ALTER ? ? ?",
				clause.Table{Name: stmt.Table}, clause.Column{Name: oldName},
				clause.Column{Name: newName}, m.FullDataTypeOf(field),
			).Error
		}

		return fmt.Errorf("failed to look up field with name: %s", newName)
	})
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		err := m.DropIndex(value, oldName)
		if err != nil {
			return err
		}

		if idx := stmt.Schema.LookIndex(newName); idx == nil {
			if idx = stmt.Schema.LookIndex(oldName); idx != nil {
				opts := m.BuildIndexOptions(idx.Fields, stmt)
				values := []interface{}{clause.Column{Name: newName}, clause.Table{Name: stmt.Table}, opts}

				createIndexSQL := "CREATE "
				if idx.Class != "" {
					createIndexSQL += idx.Class + " "
				}
				createIndexSQL += "INDEX ? ON ??"

				if idx.Type != "" {
					createIndexSQL += " USING " + idx.Type
				}

				return m.DB.Exec(createIndexSQL, values...).Error
			}
		}

		return m.CreateIndex(value, newName)
	})

}

func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	tx := m.DB.Session(&gorm.Session{})
	tx.Exec("SET FOREIGN_KEY_CHECKS = 0;")
	for i := len(values) - 1; i >= 0; i-- {
		if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
			return tx.Exec("DROP TABLE IF EXISTS ? CASCADE", clause.Table{Name: stmt.Table}).Error
		}); err != nil {
			return err
		}
	}
	tx.Exec("SET FOREIGN_KEY_CHECKS = 1;")
	return nil
}

func (m Migrator) DropConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, chk, table := m.GuessConstraintAndTable(stmt, name)
		if chk != nil {
			return m.DB.Exec("ALTER TABLE ? DROP CHECK ?", clause.Table{Name: stmt.Table}, clause.Column{Name: chk.Name}).Error
		}
		if constraint != nil {
			name = constraint.Name
		}

		return m.DB.Exec(
			"ALTER TABLE ? DROP FOREIGN KEY ?", clause.Table{Name: table}, clause.Column{Name: name},
		).Error
	})
}

// ColumnTypes column types return columnTypes,error
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var
		columnTypeSQL   = "SELECT B.RDB$FIELD_NAME column_name, B.RDB$NULL_FLAG is_nullable, (CASE D.RDB$TYPE_NAME  WHEN 'TEXT' THEN 'CHAR' WHEN 'INT64' THEN 'BIGINT' WHEN 'LONG' THEN IIF(C.RDB$FIELD_SCALE=0,'INTEGER','NUMBERIC') WHEN 'SHORT' THEN 'SMALLINT' WHEN 'DOUBLE' THEN 'DOUBLE' WHEN 'VARYING' THEN 'VARCHAR' WHEN 'FLOAT' THEN 'FLOAT' WHEN 'BLOB' THEN 'BLOB' WHEN 'TIMESTAMP' THEN 'TIMESTAMP' END) data_type, IIF(D.RDB$TYPE_NAME='VARYING',C.RDB$FIELD_LENGTH/4,C.RDB$FIELD_LENGTH*4) character_maximum_length, C.RDB$FIELD_PRECISION numeric_precision, C.RDB$FIELD_SCALE numeric_scale FROM RDB$RELATIONS A INNER JOIN RDB$RELATION_FIELDS B ON A.RDB$RELATION_NAME = B.RDB$RELATION_NAME INNER JOIN RDB$FIELDS C ON B.RDB$FIELD_SOURCE = C.RDB$FIELD_NAME INNER JOIN RDB$TYPES D ON C.RDB$FIELD_TYPE = D.RDB$TYPE WHERE A.RDB$SYSTEM_FLAG = 0 AND D.RDB$FIELD_NAME = 'RDB$FIELD_TYPE' AND A.RDB$RELATION_NAME = 'USERS' ORDER BY B.RDB$FIELD_POSITION "

		columns, rowErr := m.DB.Raw(columnTypeSQL, strings.ToUpper(stmt.Table)).Rows()
		if rowErr != nil {
			return rowErr
		}

		defer columns.Close()

		for columns.Next() {
			var column Column
			var values = []interface{}{&column.name, &column.nullable, &column.datatype,
				&column.maxLen, &column.precision, &column.scale}

			if scanErr := columns.Scan(values...); scanErr != nil {
				return scanErr
			}
			columnTypes = append(columnTypes, column)
		}
		return nil
	})

	return columnTypes, err
}

func (m Migrator) HasTable(value interface{}) bool {
	var count int64

	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw("SELECT COUNT(*) FROM RDB$RELATIONS WHERE RDB$FLAGS=1 and RDB$RELATION_NAME = ?", strings.ToUpper(stmt.Table)).Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Raw(
		"select count(*) from RDB$INDICES where RDB$RELATION_NAME= ? and RDB$INDEX_NAME= ?",
			strings.ToUpper(stmt.Table), strings.ToUpper(name),
		).Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, chk, _ := m.GuessConstraintAndTable(stmt, name)
		if constraint != nil {
			name = constraint.Name
		} else if chk != nil {
			name = chk.Name
		}

		return m.DB.Raw(
			"select count(*) from RDB$RELATION_CONSTRAINTS where RDB$RELATION_NAME= ? and RDB$CONSTRAINT_NAME = ?",
			strings.ToUpper(stmt.Table), strings.ToUpper(name),
		).Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		name := field
		if field := stmt.Schema.LookUpField(field); field != nil {
			name = field.DBName
		}

		return m.DB.Raw(
			"SELECT count(*) FROM RDB$RELATION_FIELDS WHERE RDB$RELATION_NAME = ? AND RDB$FIELD_NAME = ?",
			strings.ToUpper(stmt.Table), strings.ToUpper(name),
		).Row().Scan(&count)
	})

	return count > 0
}