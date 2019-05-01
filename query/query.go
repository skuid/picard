/*
Package query helps maintain aliases to each table so that when joins and
columns are added they can be properly aliased.
*/
package query

import (
	"fmt"
	"strings"

	sql "github.com/Masterminds/squirrel"
)

const (
	aliasedField string = "%[1]v.%[2]v"
	aliasedCol   string = "%[1]v.%[2]v AS \"%[1]v.%[2]v\""
	aliasedJoin  string = "%[2]v AS %[1]v ON %[3]v = %[4]v"
)

/*
Table represents a select, and is the root of the structure. Start here to build
a query by calling
	tbl := New("my_table")
*/
type Table struct {
	Counter int
	Alias   string
	Name    string
	columns []string
	Joins   []Join
	Wheres  []Where
}

/*
Join holds a very simple join definition, including a pointer to the parent table
and the joined table and type of join
*/
type Join struct {
	Type        string
	ParentField string
	JoinField   string
	Table       *Table
}

/*
Where holds a very simple where clause, and will result in an = check
*/
type Where struct {
	Field string
	Val   interface{}
}

/*
New returns a new table. This is a good starting point
*/
func New(name string) *Table {
	return &Table{
		Counter: 1,
		Alias:   "t0",
		Name:    name,
		columns: make([]string, 0),
	}
}

/*
AddColumns adds an array of columns to the current table, adding the aliases
*/
func (t *Table) AddColumns(cols []string) {
	t.columns = append(t.columns, cols...)
}

/*
AddWhere adds one where clause, WHERE {field} = {val}
*/
func (t *Table) AddWhere(field string, val interface{}) {
	t.Wheres = append(t.Wheres, Where{
		Field: field,
		Val:   val,
	})
}

/*
AppendJoin adds a join with the proper aliasing, including any columns requested
from that table
*/
func (t *Table) AppendJoin(tbl, joinField, parentField, jType string) *Table {
	alias := fmt.Sprintf("t%d", t.Counter)
	t.Counter++

	parentAlias := t.Alias

	if len(t.Joins) > 0 {
		parentAlias = t.Joins[len(t.Joins)-1].Table.Alias
	}

	join := Join{
		Table: &Table{
			Alias: alias,
			Name:  tbl,
		},
		ParentField: fmt.Sprintf(aliasedField, parentAlias, parentField),
		JoinField:   fmt.Sprintf(aliasedField, alias, joinField),
		Type:        jType,
	}

	t.Joins = append(t.Joins, join)

	return join.Table
}

/*
Columns gets the join columns including the proper alias
*/
func (t *Table) Columns() []string {
	cols := make([]string, 0, len(t.columns))

	for _, col := range t.columns {
		cols = append(cols, fmt.Sprintf(aliasedCol, t.Alias, col))
	}

	return cols
}

/*
Columns gets the join columns including the proper alias
*/
func (j *Join) Columns() []string {
	return j.Table.Columns()
}

func (j *Join) String() string {
	return fmt.Sprintf(
		aliasedJoin,
		j.Table.Alias,
		j.Table.Name,
		j.JoinField,
		j.ParentField,
	)
}

/*
FieldDescriptor holds the table/field info for an aliased field
*/
type FieldDescriptor struct {
	Table string
	Field string
}

/*
FieldAliases returns a map of all columns on a table and that table's joins.
*/
func (t *Table) FieldAliases() map[string]FieldDescriptor {
	aliasMap := make(map[string]FieldDescriptor)
	for _, col := range t.columns {
		aliasMap[fmt.Sprintf(aliasedField, t.Alias, col)] = FieldDescriptor{
			Table: t.Name,
			Field: col,
		}
	}

	for _, join := range t.Joins {
		for _, col := range join.Table.columns {
			aliasMap[fmt.Sprintf(aliasedField, join.Table.Alias, col)] = FieldDescriptor{
				Table: join.Table.Name,
				Field: col,
			}
		}
	}

	return aliasMap
}

/*
ToSQL returns the SQL statement, as it currently stands.
*/
func (t *Table) ToSQL() (string, []interface{}, error) {
	return t.BuildSQL().ToSql()
}

/*
BuildSQL returns a squirrel SelectBuilder, which can be used to execute the query
or to just add more to the query
*/
func (t *Table) BuildSQL() sql.SelectBuilder {
	bld := sql.Select(t.Columns()...).
		PlaceholderFormat(sql.Dollar).
		From(fmt.Sprintf("%s AS %s", t.Name, t.Alias))

	for _, where := range t.Wheres {
		bld = bld.Where(sql.Eq{fmt.Sprintf(aliasedField, t.Alias, where.Field): where.Val})
	}

	for _, join := range t.Joins {

		bld = bld.Columns(join.Columns()...)

		switch strings.ToLower(join.Type) {
		case "right":
			bld = bld.RightJoin(join.String())
		case "left":
			bld = bld.LeftJoin(join.String())
		default:
			bld = bld.Join(join.String())

		}

		for _, where := range join.Table.Wheres {
			bld = bld.Where(sql.Eq{fmt.Sprintf(aliasedField, join.Table.Alias, where.Field): where.Val})
		}
	}

	return bld
}
