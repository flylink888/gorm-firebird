package firebird

import (
	"gorm.io/gorm/schema"
	"strings"
)

type NamingStrategy struct {
	schema.NamingStrategy
}

func (ns NamingStrategy) TableName(str string) string {
	return strings.ToUpper(ns.NamingStrategy.TableName(str))
}

func (ns NamingStrategy) ColumnName(table, column string) string {
	return strings.ToUpper(ns.NamingStrategy.ColumnName(table,column))
}

func (ns NamingStrategy) JoinTableName(table string) (name string) {
	return strings.ToUpper(ns.NamingStrategy.JoinTableName(table))
}

func (ns NamingStrategy) RelationshipFKName(relationship schema.Relationship) (name string) {
	return strings.ToUpper(ns.NamingStrategy.RelationshipFKName(relationship))
}

func (ns NamingStrategy) CheckerName(table, column string) (name string) {
	return strings.ToUpper(ns.NamingStrategy.CheckerName(table, column))
}

func (ns NamingStrategy) IndexName(table, column string) (name string) {
	return strings.ToUpper(ns.NamingStrategy.IndexName(table, column))
}
