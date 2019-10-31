package queryparts

import (
	"fmt"
)

const (
	aliasedJoin string = "%[2]v AS %[1]v ON"
)

/*
Join holds a very simple join definition, including a pointer to the parent table
and the joined table and type of join
*/
type Join struct {
	Type        string
	Parent      *Table
	ParentField string
	JoinField   string
	Table       *Table
}

/*
Columns gets the join columns including the proper alias
*/
func (j *Join) Columns() []string {
	return j.Table.Columns()
}

/*
Build will turn the current join into the needed SQL, adding all of the aliases
to the join. It will return something like:
	t1.customer ON t1.parent_id = t0.id
*/
func (j *Join) Build(parentAlias string) string {
	return fmt.Sprintf(
		aliasedJoin,
		j.Table.Alias,
		j.Table.Name,
	)
}