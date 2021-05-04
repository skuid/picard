package queryparts

import (
	"fmt"
	"strings"

	sql "github.com/Masterminds/squirrel"
	"github.com/skuid/picard/stringutil"
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
	root         *Table
	Alias        string
	RefPath      string
	Name         string
	columns      []string
	lookups      map[string]interface{}
	Joins        []Join
	Wheres       sql.And
	MultiTenancy sql.Eq
}

/*
New returns a new table.
*/
func New(name string) *Table {
	index := 0
	return NewAliased(name, stringutil.GenerateTableAlias(&index), "")
}

// NewAliased returns a new table with the given alias
func NewAliased(name string, alias string, refPath string) *Table {
	return &Table{
		Alias:   alias,
		RefPath: refPath,
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
func (t *Table) AddWhere(column string, val interface{}) {
	t.Wheres = append(t.Wheres, sql.Eq{fmt.Sprintf(AliasedField, t.Alias, column): val})
}

/*
AddWhereGroup adds a grouping of ORS or ANDs to the where clause
*/
func (t *Table) AddWhereGroup(group sql.Sqlizer) {
	t.Wheres = append(t.Wheres, group)
}

/*
AddMultitenancyWhere creates a multitenancy WHERE condition
*/
func (t *Table) AddMultitenancyWhere(column string, val interface{}) {
	t.MultiTenancy = sql.Eq{
		fmt.Sprintf(AliasedField, t.Alias, column): val,
	}
}

/*
AppendJoin adds a join with the proper aliasing, including any columns requested
from that table
*/
func (t *Table) AppendJoin(tbl, joinField, parentField, jType string, counter *int) *Table {
	var root *Table
	if t.root != nil {
		root = t.root
	} else {
		root = t
	}

	alias := stringutil.GenerateTableAlias(counter)

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
			Alias:   t.Alias,
			RefPath: t.RefPath,
			Table:   t.Name,
			Column:  col,
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

	if t.MultiTenancy != nil {
		bld = bld.Where(t.MultiTenancy)
	}

	for _, where := range t.Wheres {
		bld = bld.Where(where)
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

	if t.MultiTenancy != nil {
		bld = bld.Where(t.MultiTenancy)
	}

	for _, where := range t.Wheres {
		bld = bld.Where(where)
	}

	return bld
}

func sqlizeJoin(bld sql.SelectBuilder, join Join) sql.SelectBuilder {

	bld = bld.Columns(join.Columns()...)

	jc := sql.Sqlizer(sql.Expr(fmt.Sprintf(AliasedField, join.Table.Alias, join.JoinField) + " = " + fmt.Sprintf(AliasedField, join.Parent.Alias, join.ParentField)))
	if join.Table.MultiTenancy != nil {
		where := join.Table.MultiTenancy
		jc = sql.And{
			jc,
			where,
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
		bld = bld.Where(where)
	}

	for _, join := range join.Table.Joins {
		bld = sqlizeJoin(bld, join)
	}

	return bld

}
