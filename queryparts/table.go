package queryparts

import (
	"fmt"
	"strings"
	sql "github.com/Masterminds/squirrel"
)

const (
	aliasedCol string = "%[1]v.%[2]v AS \"%[1]v.%[2]v\""
)

/*
Table represents a select, and is the root of the structure. Start here to build
a query by calling
	tbl := New("my_table")
*/
type Table struct {
	root    *Table
	Counter int
	Alias   string
	Name    string
	columns []string
	lookups map[string]interface{}
	Joins   []Join
	Wheres  []Where
	MultiTenancy Where
}

/*
New returns a new table. This is a good starting point
*/
func New(name string) *Table {
	return NewIndexed(name, 0)
}

/*
NewIndexed returns a new table. This is a good starting point
*/
func NewIndexed(name string, index int) *Table {
	return &Table{
		Counter: index + 1,
		Alias:   fmt.Sprintf("t%d", index),
		Name:    name,
		columns: make([]string, 0),
		lookups: make(map[string]interface{}),
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

func (t *Table) AddMultitenancyWhere(field string, val interface{}) {
	t.MultiTenancy = Where{
		Field: field,
		Val: val,
	}
}

/*
AppendJoin adds a join with the proper aliasing, including any columns requested
from that table
*/
func (t *Table) AppendJoin(tbl, joinField, parentField, jType string) *Table {
	var root *Table
	if t.root != nil {
		root = t.root
	} else {
		root = t
	}

	alias := fmt.Sprintf("t%d", root.Counter)
	root.Counter++

	join := Join{
		Table: &Table{
			root:  root,
			Alias: alias,
			Name:  tbl,
		},
		Parent:      t,
		ParentField: parentField,
		JoinField:   joinField,
		Type:        jType,
	}

	t.Joins = append(t.Joins, join)

	return join.Table
}

/*
AppendJoinTable adds a join with the proper aliasing, including any columns requested
from that table
*/
func (t *Table) AppendJoinTable(tbl *Table, joinField, parentField, jType string) *Table {
	var root *Table
	if t.root != nil {
		root = t.root
	} else {
		root = t
	}

	// alias := fmt.Sprintf("t%d", root.Counter)
	// tbl.Alias = alias
	root.Counter++

	join := Join{
		Table:       tbl,
		Parent:      t,
		ParentField: parentField,
		JoinField:   joinField,
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
FieldAliases returns a map of all columns on a table and that table's joins.
*/
func (t *Table) FieldAliases() map[string]FieldDescriptor {
	aliasMap := make(map[string]FieldDescriptor)
	for _, col := range t.columns {
		aliasMap[fmt.Sprintf(AliasedField, t.Alias, col)] = FieldDescriptor{
			Alias: t.Alias,
			Table: t.Name,
			Field: col,
		}
	}

	for _, join := range t.Joins {
		jmap := join.Table.FieldAliases()
		for key, val := range jmap {
			aliasMap[key] = val
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

	if t.MultiTenancy != (Where{}) {
		bld = bld.Where(
			sql.Eq{
				fmt.Sprintf(AliasedField, t.Alias, t.MultiTenancy.Field):
				t.MultiTenancy.Val,
			},
		)
	}

	for _, where := range t.Wheres {
		bld = bld.Where(sql.Eq{fmt.Sprintf(AliasedField, t.Alias, where.Field): where.Val})
	}

	for _, join := range t.Joins {
		bld = sqlizeJoin(bld, join)
	}

	return bld
}

/*
DeleteSQL returns a squirrel SelectBuilder, which can be used to execute the query
or to just add more to the query
*/
func (t *Table) DeleteSQL() sql.DeleteBuilder {
	bld := sql.Delete(fmt.Sprintf("%s AS %s", t.Name, t.Alias)).
		PlaceholderFormat(sql.Dollar)

	if t.MultiTenancy != (Where{}) {
		bld = bld.Where(
			sql.Eq{
				fmt.Sprintf(AliasedField, t.Alias, t.MultiTenancy.Field):
				t.MultiTenancy.Val,
			},
		)
	}

	for _, where := range t.Wheres {
		bld = bld.Where(sql.Eq{fmt.Sprintf(AliasedField, t.Alias, where.Field): where.Val})
	}

	return bld
}

func sqlizeJoin(bld sql.SelectBuilder, join Join) sql.SelectBuilder {

	bld = bld.Columns(join.Columns()...)

	jc := sql.Sqlizer(sql.Expr(fmt.Sprintf(AliasedField, join.Table.Alias, join.JoinField) + " = " + fmt.Sprintf(AliasedField, join.Parent.Alias, join.ParentField)))
	if join.Table.MultiTenancy != (Where{}) {
		where := join.Table.MultiTenancy
		jc = sql.And{
			jc,
			sql.Eq{
				fmt.Sprintf(AliasedField, join.Table.Alias, where.Field):
				where.Val,
			},
		}
	}

	switch strings.ToLower(join.Type) {
	case "right":
		bld = bld.RightJoin(join.Build(join.Parent.Alias))
	case "left":
		bld = bld.LeftJoin(join.Build(join.Parent.Alias))
	default:
		bld = bld.Join(join.Build(join.Parent.Alias))

	}
	bld = bld.JoinClause(jc)


	for _, where := range join.Table.Wheres {
		bld = bld.Where(sql.Eq{fmt.Sprintf(AliasedField, join.Table.Alias, where.Field): where.Val})
	}

	for _, join := range join.Table.Joins {
		bld = sqlizeJoin(bld, join)
	}

	return bld

}