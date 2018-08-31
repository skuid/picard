package entity

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/skuid/warden/pkg/mapvalue"
)

func deRefSubConditions(fldLoader fieldDefLoader, conditions []map[string]interface{}) (map[string]interface{}, error) {

	refedSubCs := make(map[string]subCondition)
	jEnts := make([]joinEntity, 0)

	// Get a list of joins we need to add as conditions and the referenced
	// subConditions, keyed by the joined table's name
	for _, c := range conditions {
		tbl := mapvalue.String(c, "joinObject")
		subCs := mapvalue.MapSlice(c, "subConditions")
		jes, refedSubs := getRefedSubConditions(tbl, subCs)

		// Join refedSubs to refedSubCs (map key match, with append of subCs)
		jEnts = append(jEnts, jes...)

		for refField, subC := range refedSubs {
			refedSubCs[refField] = subCondition{
				entity:     subC.entity,
				joins:      subC.joins,
				conditions: subC.conditions,
			}
		}
	}

	// Now we have a list of subs that were references and a list of join
	// definitions. If we have no sub refs, we can bail.
	if len(jEnts) <= 0 {
		return nil, nil
	}

	// Get schema information to resolve join references
	joins, err := fldLoader(jEnts)
	if err != nil {
		return nil, err
	}

	keyedJoinDefs := make(map[string]map[string]interface{})
	// Now to add our joins.
	for _, jDef := range joins {
		key := fmt.Sprintf("%s.%s", jDef.parent.entity, jDef.parent.field)
		keyedJoinDefs[key] = createJoinCondition(jDef)
	}

	refeds := make(map[string]interface{})
	for field, subC := range refedSubCs {
		conditions := make([]map[string]interface{}, len(subC.joins))

		for i, joinKey := range subC.joins {
			conditions[i] = copyMap(keyedJoinDefs[joinKey])
		}

		parentTbl := conditions[0]["sourceObject"]
		parentFld := conditions[0]["field"]
		delete(conditions[0], "sourceObject")

		conditions[len(conditions)-1]["subConditions"] = subC.conditions
		conditions[len(conditions)-1]["subConditionLogic"] = getLogic(len(subC.conditions))

		ic := make([]interface{}, len(conditions))
		for i, c := range conditions {
			ic[i] = c
		}

		refeds[field] = map[string]interface{}{
			"object":     parentTbl,
			"field":      parentFld,
			"conditions": ic,
		}
	}

	return refeds, nil
}

type joinEntity struct {
	field  string
	entity string
}

type joinDefinition struct {
	parent joinEntity
	join   joinEntity
}

type subCondition struct {
	entity     string
	logic      string
	joins      []string
	conditions []interface{}
}

/*
getRefedSubConditions will take the parent table, the parent's subConditionLogic,
and the array of subConditions (for one condition). It will return two things:

[]joinEntity - slice containing an array of join entities, which hold information
about each join that needs to be performed. Each join entity will contain:
- entity
- field

map[string]subCondition - This complicated beast contains a hash map, where the
key is the table name (entity) these sub conditions should be attached to. The
subCondition value holds the logic that came from the parent, along with an array
of all of the sub conditions that belong to this (lower depth, post joins) table
*/
func getRefedSubConditions(
	parentTable string,
	subConditions []map[string]interface{},
) ([]joinEntity, map[string]subCondition) {

	refedSubCs := make(map[string]subCondition)
	refedEntities := make([]joinEntity, 0)

	for _, subC := range subConditions {
		fto := mapvalue.String(subC, "fieldTargetObjects")

		if fto != "" {
			field := mapvalue.String(subC, "field")
			parts := strings.Split(field, ".")

			if len(parts) > 1 {
				newSubC := copyMap(subC)

				delete(newSubC, "fieldTargetObjects")
				newSubC["field"], parts = pop(parts)
				joinEnts := refedFieldToEntities(parentTable, parts)
				lastTbl := joinEnts[len(joinEnts)-1].entity

				joinEntsKeys := make([]string, len(joinEnts))
				for i, je := range joinEnts {
					key := fmt.Sprintf("%s.%s", je.entity, je.field)
					joinEntsKeys[i] = key
				}

				if len(refedSubCs[field].conditions) <= 0 {
					refedSubCs[field] = subCondition{
						entity: lastTbl,
						joins:  joinEntsKeys,
						conditions: []interface{}{
							newSubC,
						},
					}
				} else {
					tblCs := refedSubCs[field].conditions
					refedSubCs[field] = subCondition{
						entity:     lastTbl,
						joins:      joinEntsKeys,
						conditions: append(tblCs, newSubC),
					}
				}

				refedEntities = append(refedEntities, joinEnts...)
			}
		}
	}

	return refedEntities, refedSubCs
}

func pop(slice []string) (string, []string) {
	return slice[len(slice)-1], slice[:len(slice)-1]
}

func getLogic(n int) string {
	a := make([]string, n)
	for i := range a {
		a[i] = fmt.Sprintf("%d", i+1)
	}
	return strings.Join(a, " AND ")
}

/*
refedFieldToEntities splits an array of parts (split by "." typically), and turns
them into an array of join entities.
*/
func refedFieldToEntities(parentTbl string, parts []string) []joinEntity {
	entities := make([]joinEntity, 0)
	lastTbl := parentTbl
	for _, jc := range parts {
		field := strings.TrimSuffix(jc, "__rel")
		tbl := strings.TrimSuffix(field, "_id")

		entities = append(entities, joinEntity{
			field:  field,
			entity: lastTbl,
		})
		lastTbl = tbl
	}
	return entities
}

type fieldDefLoader func(entities []joinEntity) ([]joinDefinition, error)

/*
loadFieldDefs looks in the persistence store to pull out the join information
for each parent field, returning a full set of join information

The lookup should look something like this:

	SELECT
		o.name,
		f.name,
		f.reference_to
	FROM
		data_source_object as o
	LEFT JOIN
		data_source_field as f on o.id = f.data_source_object_id
	WHERE
		o.organization_id = '558529cb-7b66-4658-9488-c5852ceb289b' AND
		o.data_source_id = '49d60591-9e38-415b-a292-18809f2541bc' AND
		(
			(o.name = 'store' AND f.name = 'address_id') OR
			(o.name = 'address' AND f.name = 'city_id')
		)

It will return a slice of join definitions, each of which contains

	{
		parent: {
			entity,
			field
		},
		join: {
			entity,
			field
		}
	}

Each of of these will be used later to map a join from the parent's entity.field
to the sub (joined) table's entity.field
*/
func getFieldLoader(db *sql.DB, orgID string, dsID string) fieldDefLoader {
	return func(entities []joinEntity) ([]joinDefinition, error) {

		defs := make([]joinDefinition, len(entities))

		if len(entities) <= 0 {
			return defs, nil
		}

		qb := sq.StatementBuilder.
			PlaceholderFormat(sq.Dollar).
			RunWith(db)

		// Build the "where" clause for the query. First we need to filter by
		// orgID and datasourceID. Then we need to add each entity/field
		// combination
		wheres := sq.And{
			sq.Eq{"o.organization_id": orgID},
			sq.Eq{"o.data_source_id": dsID},
		}

		ors := sq.Or{}

		for _, e := range entities {
			ors = append(ors, sq.And{
				sq.Eq{
					"o.name": e.entity,
				},
				sq.Eq{
					"f.name": e.field,
				},
			})
		}

		// Build the rest of the query and execute
		q := qb.
			Select([]string{
				"o.name AS entity",
				"f.name AS field",
				"f.reference_to AS ref_to",
			}...).
			From("data_source_object AS o").
			LeftJoin("data_source_field AS f ON o.id = f.data_source_object_id").
			Where(append(wheres, ors))

		rows, err := q.Query()
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		i := 0
		var (
			entity string
			field  string
			refTo  []byte
		)

		// For each row, turn the information into a joinDefinition
		for rows.Next() {
			err := rows.Scan(&entity, &field, &refTo)
			if err != nil {
				return nil, err
			}
			var refs []referenceEntity
			json.Unmarshal(refTo, &refs)

			if len(refs) > 1 || len(refs) < 1 {
				return nil, fmt.Errorf(
					"We should have gotten exactly one reference. Found %d references for entity: %s, field: %s",
					len(refs),
					entity,
					field,
				)
			}

			defs[i] = buildJoinDefinition(entity, field, refs[0])
			i++
		}

		return defs, nil
	}
}

func buildJoinDefinition(entity string, field string, ref referenceEntity) joinDefinition {
	return joinDefinition{
		parent: joinEntity{
			entity: entity,
			field:  field,
		},
		join: joinEntity{
			entity: ref.Entity,
			field:  ref.Field,
		},
	}
}

type referenceEntity struct {
	Field  string `json:"keyfield"`
	Entity string `json:"object"`
}

func createJoinCondition(jDef joinDefinition) map[string]interface{} {
	return map[string]interface{}{
		"encloseValueInQuotes": false,
		"field":                jDef.parent.field,
		"sourceObject":         jDef.parent.entity,
		"joinField":            jDef.join.field,
		"joinObject":           jDef.join.entity,
		"operator":             "in",
		"inactive":             false,
		"originalInactive":     false,
		"sourceType":           "fieldvalue",
		"type":                 "join",
		"originalValue":        nil,
		"value":                nil,
	}
}

func copyMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})

	for k, v := range in {
		out[k] = v
	}

	return out
}
