// Copyright 2018 Huan Du. All rights reserved.
// Licensed under the MIT license that can be found in the LICENSE file.

package sqlbuilder

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	insertMarkerInit injectionMarker = iota
	insertMarkerAfterInsertInto
	insertMarkerAfterCols
	insertMarkerAfterValues
)

// NewInsertBuilder creates a new INSERT builder.
func NewInsertBuilder() *InsertBuilder {
	return DefaultFlavor.NewInsertBuilder()
}

func newInsertBuilder() *InsertBuilder {
	args := &Args{}
	return &InsertBuilder{
		verb:      "INSERT",
		args:      args,
		injection: newInjection(),
	}
}

// InsertBuilder is a builder to build INSERT.
type InsertBuilder struct {
	verb   string
	table  string
	cols   []string
	values [][]string

	args *Args

	injection *injection
	marker    injectionMarker
}

var _ Builder = new(InsertBuilder)

// InsertInto sets table name in INSERT.
func InsertInto(table string) *InsertBuilder {
	return DefaultFlavor.NewInsertBuilder().InsertInto(table)
}

// InsertInto sets table name in INSERT.
func (ib *InsertBuilder) InsertInto(table string) *InsertBuilder {
	ib.table = Escape(table)
	ib.marker = insertMarkerAfterInsertInto
	return ib
}

// InsertIgnoreInto sets table name in INSERT IGNORE.
func InsertIgnoreInto(table string) *InsertBuilder {
	return DefaultFlavor.NewInsertBuilder().InsertIgnoreInto(table)
}

// InsertIgnoreInto sets table name in INSERT IGNORE.
func (ib *InsertBuilder) InsertIgnoreInto(table string) *InsertBuilder {
	ib.args.Flavor.PrepareInsertIgnore(table, ib)
	return ib
}

// ReplaceInto sets table name and changes the verb of ib to REPLACE.
// REPLACE INTO is a MySQL extension to the SQL standard.
func ReplaceInto(table string) *InsertBuilder {
	return DefaultFlavor.NewInsertBuilder().ReplaceInto(table)
}

// ReplaceInto sets table name and changes the verb of ib to REPLACE.
// REPLACE INTO is a MySQL extension to the SQL standard.
func (ib *InsertBuilder) ReplaceInto(table string) *InsertBuilder {
	ib.verb = "REPLACE"
	ib.table = Escape(table)
	ib.marker = insertMarkerAfterInsertInto
	return ib
}

func (ib *InsertBuilder) InsertItem(item interface{}) *InsertBuilder {
	var cols []string
	var values []interface{}

	valueType := reflect.TypeOf(item)
	valueData := reflect.ValueOf(item)

	// 检查是否是指针类型
	if valueType.Kind() == reflect.Ptr {
		// 获取指针所指向的实际类型
		valueType = valueType.Elem()
		valueData = valueData.Elem()
	}

	// 构建插入语句
	for i := 0; i < valueType.NumField(); i++ {
		dbTag := valueType.Field(i).Tag.Get("db")
		dbList := strings.Split(dbTag, ";")
		// if primary_key in dbTag, skip
		if len(dbList) > 1 && dbList[1] == "primary_key" {
			continue
		}
		if len(dbList) > 0 {
			cols = append(cols, dbList[0])
			values = append(values, valueData.Field(i).Interface())
		}
	}
	ib.Cols(cols...).Values(values...)
	return ib
}

// Cols sets columns in INSERT.
func (ib *InsertBuilder) Cols(col ...string) *InsertBuilder {
	ib.cols = EscapeAll(col...)
	ib.marker = insertMarkerAfterCols
	return ib
}

// Values adds a list of values for a row in INSERT.
func (ib *InsertBuilder) Values(value ...interface{}) *InsertBuilder {
	placeholders := make([]string, 0, len(value))

	for _, v := range value {
		placeholders = append(placeholders, ib.args.Add(v))
	}

	ib.values = append(ib.values, placeholders)
	ib.marker = insertMarkerAfterValues
	return ib
}

// String returns the compiled INSERT string.
func (ib *InsertBuilder) String() string {
	s, _ := ib.Build()
	return s
}

// Build returns compiled INSERT string and args.
// They can be used in `DB#Query` of package `database/sql` directly.
func (ib *InsertBuilder) Build() (sql string, args []interface{}) {
	return ib.BuildWithFlavor(ib.args.Flavor)
}

// BuildWithFlavor returns compiled INSERT string and args with flavor and initial args.
// They can be used in `DB#Query` of package `database/sql` directly.
func (ib *InsertBuilder) BuildWithFlavor(flavor Flavor, initialArg ...interface{}) (sql string, args []interface{}) {
	buf := newStringBuilder()
	ib.injection.WriteTo(buf, insertMarkerInit)

	if len(ib.table) > 0 {
		buf.WriteLeadingString(ib.verb)
		buf.WriteString(" INTO ")
		buf.WriteString(ib.table)
	}

	ib.injection.WriteTo(buf, insertMarkerAfterInsertInto)

	if len(ib.cols) > 0 {
		buf.WriteLeadingString("(")
		buf.WriteString(strings.Join(ib.cols, ", "))
		buf.WriteString(")")

		ib.injection.WriteTo(buf, insertMarkerAfterCols)
	}

	if len(ib.values) > 0 {
		buf.WriteLeadingString("VALUES ")
		values := make([]string, 0, len(ib.values))

		for _, v := range ib.values {
			values = append(values, fmt.Sprintf("(%v)", strings.Join(v, ", ")))
		}

		buf.WriteString(strings.Join(values, ", "))
	}

	ib.injection.WriteTo(buf, insertMarkerAfterValues)

	return ib.args.CompileWithFlavor(buf.String(), flavor, initialArg...)
}

// SetFlavor sets the flavor of compiled sql.
func (ib *InsertBuilder) SetFlavor(flavor Flavor) (old Flavor) {
	old = ib.args.Flavor
	ib.args.Flavor = flavor
	return
}

// Var returns a placeholder for value.
func (ib *InsertBuilder) Var(arg interface{}) string {
	return ib.args.Add(arg)
}

// SQL adds an arbitrary sql to current position.
func (ib *InsertBuilder) SQL(sql string) *InsertBuilder {
	ib.injection.SQL(ib.marker, sql)
	return ib
}
