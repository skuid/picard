package query

import (
	"testing"

	qp "github.com/skuid/picard/queryparts"
	"github.com/skuid/picard/testdata"
	"github.com/stretchr/testify/assert"
)

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
			testdata.FmtSQL(`
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
			testdata.FmtSQL(`
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
			testdata.FmtSQL(`
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
	joinMt      whereTest
	joins       []joinTest
}

func TestQueryJoins(t *testing.T) {

	testCases := []struct {
		desc         string
		cols         []string
		joins        []joinTest
		tblMt        whereTest
		wheres       []whereTest
		expected     string
		expectedArgs []interface{}
	}{{
		"should create the proper SQL for a simple table select with columns and two joins",
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
			{
				tbl:         "table_b",
				joinField:   "my_id",
				parentField: "col_two",
				jType:       "",
			},
		},
		whereTest{},
		nil,
		testdata.FmtSQL(`
			SELECT t0.col_one AS "t0.col_one",
				t0.col_two AS "t0.col_two",
				t0.col_three AS "t0.col_three"
			FROM foo AS t0
			JOIN table_b AS t1 ON t1.my_id = t0.col_two
			JOIN table_b AS t2 ON t2.my_id = t0.col_two
		`),
		nil,
	},
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
					joinMt: whereTest{
						field: "col_mt",
						val:   "12345",
					},
					jType: "",
				},
			},
			whereTest{
				field: "tbl_mt",
				val:   "12345",
			},
			nil,
			testdata.FmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
				JOIN table_b AS t1 ON (t1.my_id = t0.col_two AND t1.col_mt = $1)
				WHERE t0.tbl_mt = $2
			`),
			[]interface{}{
				"12345",
				"12345",
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
				},
				{
					tbl:         "table_c",
					joinField:   "table_b_id",
					parentField: "id",
					jType:       "right",
				},
			},
			whereTest{},
			[]whereTest{
				{
					field: "col_four",
					val:   "something col 4",
				},
			},
			testdata.FmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three"
				FROM foo AS t0
				LEFT JOIN table_b AS t1 ON t1.my_id = t0.col_two
				RIGHT JOIN table_c AS t2 ON t2.table_b_id = t0.id
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
			whereTest{},
			[]whereTest{
				{
					field: "col_four",
					val:   "something col 4",
				},
			},
			testdata.FmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three",
					t1.b_col_one AS "t1.b_col_one",
					t1.b_col_two AS "t1.b_col_two",
					t2.c_col_one AS "t2.c_col_one",
					t2.c_col_two AS "t2.c_col_two"
				FROM foo AS t0
				LEFT JOIN table_b AS t1 ON t1.my_id = t0.col_two
				LEFT JOIN table_c AS t2 ON t2.table_b_id = t0.id
				WHERE t0.col_four = $1 AND
					t1.b_col_three = $2
			`),
			[]interface{}{
				"something col 4",
				333333,
			},
		},
		{
			"should create the proper SQL for a simple table select with columns and deep joins",
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
					joins: []joinTest{
						{
							tbl:         "table_d",
							joinField:   "tbl_d_id",
							parentField: "tbl_b_id",
							jType:       "left",
							cols: []string{
								"d_col_one",
								"d_col_two",
							},
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
			whereTest{},
			[]whereTest{
				{
					field: "col_four",
					val:   "something col 4",
				},
			},
			testdata.FmtSQL(`
				SELECT t0.col_one AS "t0.col_one",
					t0.col_two AS "t0.col_two",
					t0.col_three AS "t0.col_three",
					t1.b_col_one AS "t1.b_col_one",
					t1.b_col_two AS "t1.b_col_two",
					t2.d_col_one AS "t2.d_col_one",
					t2.d_col_two AS "t2.d_col_two",
					t3.c_col_one AS "t3.c_col_one",
					t3.c_col_two AS "t3.c_col_two"
				FROM foo AS t0
				LEFT JOIN table_b AS t1 ON t1.my_id = t0.col_two
				LEFT JOIN table_d AS t2 ON t2.tbl_d_id = t1.tbl_b_id
				LEFT JOIN table_c AS t3 ON t3.table_b_id = t0.id
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
			counter := 1
			assert := assert.New(t)

			tbl := New("foo")
			tbl.AddColumns(tc.cols)

			if tc.tblMt != (whereTest{}) {
				tbl.AddMultitenancyWhere(tc.tblMt.field, tc.tblMt.val)
			}

			for _, jt := range tc.joins {
				appendTestJoin(tbl, jt, &counter)
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

func appendTestJoin(tbl *qp.Table, jt joinTest, counter *int) {
	joinTbl := tbl.AppendJoin(jt.tbl, jt.joinField, jt.parentField, jt.jType, counter)
	if jt.joinMt != (whereTest{}) {
		joinTbl.AddMultitenancyWhere(jt.joinMt.field, jt.joinMt.val)
	}
	joinTbl.AddColumns(jt.cols)
	for _, where := range jt.wheres {
		joinTbl.AddWhere(where.field, where.val)
	}

	for _, subJt := range jt.joins {
		appendTestJoin(joinTbl, subJt, counter)
	}

}

type fieldAliasFixture struct {
	table string
	cols  []string
	joins []fieldAliasFixture
}

func TestFieldAliases(t *testing.T) {

	testCases := []struct {
		desc     string
		fixture  fieldAliasFixture
		expected map[string]qp.FieldDescriptor
	}{
		{
			"should return an empty map if there are no columns",
			fieldAliasFixture{
				table: "table_a",
			},
			map[string]qp.FieldDescriptor{},
		},
		{
			"should generate the field aliases for a single table with no joins",
			fieldAliasFixture{
				table: "table_a",
				cols: []string{
					"col_a",
					"col_b",
				},
			},
			map[string]qp.FieldDescriptor{
				"t0.col_a": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "table_a",
					Column: "col_a",
				},
				"t0.col_b": qp.FieldDescriptor{
					Alias:  "t0",
					Table:  "table_a",
					Column: "col_b",
				},
			},
		},
		{
			"should generate the field aliases for a single table with joins",
			fieldAliasFixture{
				table: "table_a",
				cols: []string{
					"col_a",
					"col_b",
				},
				joins: []fieldAliasFixture{
					{
						table: "table_b",
						cols: []string{
							"col_c",
							"col_d",
						},
						joins: []fieldAliasFixture{
							{
								table: "table_d",
								cols: []string{
									"col_d_a",
								},
							},
						},
					},
					{
						table: "table_c",
						cols: []string{
							"col_e",
						},
					},
				},
			},
			map[string]qp.FieldDescriptor{
				"t0.col_a": {
					Alias:  "t0",
					Table:  "table_a",
					Column: "col_a",
				},
				"t0.col_b": {
					Alias:  "t0",
					Table:  "table_a",
					Column: "col_b",
				},
				"t1.col_c": {
					Alias:  "t1",
					Table:  "table_b",
					Column: "col_c",
				},
				"t1.col_d": {
					Alias:  "t1",
					Table:  "table_b",
					Column: "col_d",
				},
				"t2.col_d_a": {
					Alias:  "t2",
					Table:  "table_d",
					Column: "col_d_a",
				},
				"t3.col_e": {
					Alias:  "t3",
					Table:  "table_c",
					Column: "col_e",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert := assert.New(t)

			tbl := New(tc.fixture.table)
			tbl.AddColumns(tc.fixture.cols)

			counter := 1
			appendTestAliasJoin(tbl, tc.fixture.joins, &counter)
			actual := tbl.FieldAliases()

			assert.Equal(tc.expected, actual, "Expected the resulting SQL to match expected")
		})
	}
}

func appendTestAliasJoin(tbl *qp.Table, joins []fieldAliasFixture, counter *int) {
	for _, join := range joins {
		joinTbl := tbl.AppendJoin(join.table, "foo", "bar", "", counter)
		joinTbl.AddColumns(join.cols)
		appendTestAliasJoin(tbl, join.joins, counter)
	}
}
