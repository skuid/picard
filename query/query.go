/*
Package query helps maintain aliases to each table so that when joins and
columns are added they can be properly aliased.
*/
package query

import (
	qp "github.com/skuid/picard/queryparts"
)

/*
New returns a new table. This is a good starting point
*/
func New(name string) *qp.Table {
	return qp.NewIndexed(name, 0, "")
}

/*
NewIndexed returns a new table. This is a good starting point
*/
func NewIndexed(name string, index int, refPath string) *qp.Table {
	return qp.NewIndexed(name, index, refPath)
}
