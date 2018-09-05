package entity

import (
	"database/sql/driver"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func loadTestJSON(name string, v interface{}) error {
	jsonf, err := os.Open(name)

	if err != nil {
		return err
	}

	defer jsonf.Close()

	data, _ := ioutil.ReadAll(jsonf)

	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}

	return nil
}

func mockLoader(entities []joinEntity) ([]joinDefinition, error) {
	jd := make([]joinDefinition, len(entities))
	for i, e := range entities {
		jent := strings.TrimSuffix(e.field, "_id")
		jd[i] = joinDefinition{
			parent: joinEntity{
				entity: e.entity,
				field:  e.field,
			},
			join: joinEntity{
				entity: jent,
				field:  e.field,
			},
		}
	}
	return jd, nil
}

type expected struct {
	Conditions []map[string]interface{} `json:"conditions"`
	Metadata   map[string]interface{}   `json:"metadata"`
}

func TestDeRefSubConditions(t *testing.T) {
	cases := []struct {
		desc         string
		fixtureFile  string
		expectedFile string
	}{
		{
			"Should return the same value if there is not dereferencing to be done",
			"conditions_test/no_subreferences.fixture.json",
			"",
		},
		{
			"Should expand joins at least one level deep",
			"conditions_test/single_level_deep.fixture.json",
			"conditions_test/single_level_deep.expected.json",
		},
		{
			"Should expand joins multiple levels deep",
			"conditions_test/multiple_levels_deep.fixture.json",
			"conditions_test/multiple_levels_deep.expected.json",
		},
		{
			"Should have the right subcondition logic on the metadata if there are multiple subconditions",
			"conditions_test/multiple_subconditions.fixture.json",
			"conditions_test/multiple_subconditions.expected.json",
		},
		{
			"Should handle multiple reference subconditions",
			"conditions_test/multiple_ref_subconditions.fixture.json",
			"conditions_test/multiple_ref_subconditions.expected.json",
		},
		{
			"Should get metadata for non-id referenced fields",
			"conditions_test/multiple_deep_not_id.fixture.json",
			"conditions_test/multiple_deep_not_id.expected.json",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)

			var err error
			var fixture []map[string]interface{}
			err = loadTestJSON(c.fixtureFile, &fixture)
			if err != nil {
				assert.FailNow("Could not load fixture JSON from disk", "File: '%s'.\nError: %v", c.fixtureFile, err)
			}

			var expected map[string]interface{}
			if c.expectedFile != "" {
				err = loadTestJSON(c.expectedFile, &expected)
				if err != nil {
					assert.FailNow("Could not load expected JSON from disk", "File: '%s'.]Error: %v", c.expectedFile, err)
				}
			}

			actual, err := deRefSubConditions(mockLoader, fixture)

			assert.Empty(err, "There shouldn't be an error")
			assert.EqualValues(expected, actual)
		})
	}
}

func TestLoadFieldDefs(t *testing.T) {
	queryMatch := `(?i)^SELECT o.name AS entity, f.name AS field, f.reference_to AS ref_to FROM data_source_object AS o (.+)`
	orgID := "558529cb-7b66-4658-9488-c5852ceb289b"
	dsID := "49d60591-9e38-415b-a292-18809f2541bc"

	cases := []struct {
		desc     string
		results  *sqlmock.Rows
		args     []driver.Value
		fixture  []joinEntity
		expected []joinDefinition
	}{
		{
			"Should return join definition with a single join",
			sqlmock.NewRows([]string{"entity", "field", "reference_to"}).
				AddRow("store", "address_id", "[{\"keyfield\": \"address_id\", \"object\": \"address\"}]"),
			[]driver.Value{"store", "address_id"},
			[]joinEntity{
				{
					entity: "store",
					field:  "address_id",
				},
			},
			[]joinDefinition{
				{
					parent: joinEntity{
						entity: "store",
						field:  "address_id",
					},
					join: joinEntity{
						entity: "address",
						field:  "address_id",
					},
				},
			},
		},
		{
			"Should return join definitions with multiple joins",
			sqlmock.NewRows([]string{"entity", "field", "reference_to"}).
				AddRow("store", "address_id", "[{\"keyfield\": \"address_id\", \"object\": \"address\"}]").
				AddRow("address", "city_id", "[{\"keyfield\": \"city_id\", \"object\": \"city\"}]"),
			[]driver.Value{"store", "address_id", "address", "city_id"},
			[]joinEntity{
				{
					entity: "store",
					field:  "address_id",
				},
				{
					entity: "address",
					field:  "city_id",
				},
			},
			[]joinDefinition{
				{
					parent: joinEntity{
						entity: "store",
						field:  "address_id",
					},
					join: joinEntity{
						entity: "address",
						field:  "address_id",
					},
				},
				{
					parent: joinEntity{
						entity: "address",
						field:  "city_id",
					},
					join: joinEntity{
						entity: "city",
						field:  "city_id",
					},
				},
			},
		},
		{
			"Should return an empty join definition if there are no joins",
			sqlmock.NewRows([]string{"entity", "field", "ref_to"}),
			[]driver.Value{},
			[]joinEntity{},
			[]joinDefinition{},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)

			db, mock, err := sqlmock.New()
			if err != nil {
				assert.FailNow("Failed to create the mock", "Error: %v", err)
			}
			defer db.Close()

			if len(c.fixture) > 0 {
				mock.ExpectQuery(queryMatch).
					WithArgs(append([]driver.Value{orgID, dsID}, c.args...)...).
					WillReturnRows(c.results)
			}

			actual, err := getFieldLoader(db, orgID, dsID)(c.fixture)

			if err := mock.ExpectationsWereMet(); err != nil {
				assert.Fail("sql mock expectations not met", "Error: %v", err)
			}

			assert.Empty(err, "There shouldn't be an error")
			assert.EqualValues(c.expected, actual)
		})
	}
}

func TestGetLogic(t *testing.T) {
	cases := []struct {
		desc     string
		fixture  int
		expected string
	}{
		{
			"Should return an empty string if there are no conditions",
			0,
			"",
		},
		{
			"Should return exactly one conditions",
			1,
			"1",
		},
		{
			"Should return more than one conditions",
			4,
			"1 AND 2 AND 3 AND 4",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert := assert.New(t)

			actual := getLogic(c.fixture)

			assert.EqualValues(c.expected, actual)
		})
	}
}
