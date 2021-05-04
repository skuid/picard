/*
Package query helps maintain aliases to each table so that when joins and
columns are added they can be properly aliased.
*/
package query

import (
	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/stringutil"
)

// New returns a new table with a generated alias and the given name
func New(name string) *qp.Table {
	index := 0
	return qp.NewAliased(name, stringutil.GenerateNewTableAlias(&index), "")
}

// NewAliases returns a new table with the given alias
func NewAliased(name, alias, refPath string) *qp.Table {
	return qp.NewAliased(name, alias, refPath)
}
