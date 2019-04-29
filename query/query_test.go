package query

import (
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
)

func fmtSQL(sql string) string {
	str := strings.Replace(heredoc.Doc(sql), "\n", " ", -1)
	str = strings.Replace(str, "\t", "", -1)
	return strings.Trim(str, " ")
}

func TestNewQuery(t *testing.T) {
	t.Run("should create a new table object with the proper alias", func(t *testing.T) {
		assert := assert.New(t)

		tbl := New("foo")

		assert.Equal("foo", tbl.Name)
		assert.Equal("t0", tbl.Alias)
	})
}

func TestQueryColumns(t *testing.T) {

	testCases := []struct {
		desc     string
		cols     []string
		expected string
	}{
		{
			"should create the proper SQL for a simple table select with 3 columns",
			[]string{

				"col_one",
				"col_two",
				"col_three",
			},
			fmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
			`),
		},
		{
			"should output nothing if there are not columns",
			[]string{},
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			tbl := New("foo")
			tbl.AddColumns(tc.cols)

			actual, _, _ := tbl.ToSQL()

			assert.Equal(tc.expected, actual, "Expected the resulting SQL to match expected")
		})
	}
}

type whereTest struct {
	field string
	val   interface{}
}

func TestQueryWheres(t *testing.T) {

	testCases := []struct {
		desc     string
		cols     []string
		wheres   []whereTest
		expected string
	}{
		{
			"should create the proper SQL for a simple table select with 3 columns and a where",
			[]string{

				"col_one",
				"col_two",
				"col_three",
			},
			[]whereTest{
				{
					"col_one",
					12345,
				},
			},
			fmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
				WHERE
					t0.col_one = $1
			`),
		},
		{
			"should create the proper SQL for a simple table select with 3 columns and a few wheres",
			[]string{

				"col_one",
				"col_two",
				"col_three",
			},
			[]whereTest{
				{
					"col_one",
					12345,
				},
				{
					"col_two",
					"foo_bar_test_blah",
				},
				{
					"col_three",
					"blah blah blah",
				},
			},
			fmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
				WHERE
					t0.col_one = $1 AND
					t0.col_two = $2 AND
					t0.col_three = $3
			`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			tbl := New("foo")
			tbl.AddColumns(tc.cols)

			for _, where := range tc.wheres {
				tbl.AddWhere(where.field, where.val)
			}

			actual, actualArgs, _ := tbl.ToSQL()

			assert.Equal(tc.expected, actual, "Expected the resulting SQL to match expected")

			expectedArgs := make([]interface{}, 0, len(tc.wheres))

			for _, where := range tc.wheres {
				expectedArgs = append(expectedArgs, where.val)
			}

			assert.Equal(expectedArgs, actualArgs, "Should have also passed our where clause arguments")
		})
	}
}

type joinTest struct {
	tbl         string
	joinField   string
	parentField string
	jType       string
	cols        []string
	wheres      []whereTest
}

func TestQueryJoins(t *testing.T) {

	testCases := []struct {
		desc         string
		cols         []string
		joins        []joinTest
		wheres       []whereTest
		expected     string
		expectedArgs []interface{}
	}{
		{
			"should create the proper SQL for a simple table select with columns and one join",
			[]string{

				"col_one",
				"col_two",
				"col_three",
			},
			[]joinTest{
				{
					tbl:         "table_b",
					joinField:   "my_id",
					parentField: "col_two",
					jType:       "",
				},
			},
			nil,
			fmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
				JOIN table_b AS t1 ON t1.my_id = t0.col_two
			`),
			nil,
		},
		{
			"should create the proper SQL for a simple table select with columns and multiple joins",
			[]string{

				"col_one",
				"col_two",
				"col_three",
			},
			[]joinTest{
				{
					tbl:         "table_b",
					joinField:   "my_id",
					parentField: "col_two",
					jType:       "left",
				},
				{
					tbl:         "table_c",
					joinField:   "table_b_id",
					parentField: "id",
					jType:       "left",
				},
			},
			[]whereTest{
				{
					field: "col_four",
					val:   "something col 4",
				},
			},
			fmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
				LEFT JOIN table_b AS t1 ON t1.my_id = t0.col_two
				LEFT JOIN table_c AS t2 ON t2.table_b_id = t1.id
				WHERE t0.col_four = $1
			`),
			[]interface{}{
				"something col 4",
			},
		},
		{
			"should create the proper SQL for a simple table select with columns and multiple joins",
			[]string{

				"col_one",
				"col_two",
				"col_three",
			},
			[]joinTest{
				{
					tbl:         "table_b",
					joinField:   "my_id",
					parentField: "col_two",
					jType:       "left",
					cols: []string{
						"b_col_one",
						"b_col_two",
					},
					wheres: []whereTest{
						{
							field: "b_col_three",
							val:   333333,
						},
					},
				},
				{
					tbl:         "table_c",
					joinField:   "table_b_id",
					parentField: "id",
					jType:       "left",
					cols: []string{
						"c_col_one",
						"c_col_two",
					},
				},
			},
			[]whereTest{
				{
					field: "col_four",
					val:   "something col 4",
				},
			},
			fmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three",
					t1.b_col_one AS "t1.b_col_one",
					t1.b_col_two AS "t1.b_col_two",
					t2.c_col_one AS "t2.c_col_one",
					t2.c_col_two AS "t2.c_col_two"
				FROM foo AS t0
				LEFT JOIN table_b AS t1 ON t1.my_id = t0.col_two
				LEFT JOIN table_c AS t2 ON t2.table_b_id = t1.id
				WHERE t0.col_four = $1 AND
					t1.b_col_three = $2
			`),
			[]interface{}{
				"something col 4",
				333333,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			tbl := New("foo")
			tbl.AddColumns(tc.cols)

			for _, join := range tc.joins {
				tbl := tbl.AppendJoin(join.tbl, join.joinField, join.parentField, join.jType, join.cols)
				for _, where := range join.wheres {
					tbl.AddWhere(where.field, where.val)
				}
			}

			for _, where := range tc.wheres {
				tbl.AddWhere(where.field, where.val)
			}

			actual, actualArgs, _ := tbl.ToSQL()

			assert.Equal(tc.expected, actual, "Expected the resulting SQL to match expected")

			expectedArgs := make([]interface{}, 0, len(tc.wheres))

			for _, where := range tc.wheres {
				expectedArgs = append(expectedArgs, where.val)
			}

			assert.Equal(tc.expectedArgs, actualArgs, "Should have the same arguments to the query")
		})
	}
}
