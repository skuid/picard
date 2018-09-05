package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/skuid/warden/pkg/auth"
	"github.com/skuid/warden/pkg/ds"
)

func GetErrorData(t *testing.T, s string) (message, error string) {
	var dat map[string]string
	if err := json.Unmarshal([]byte(s), &dat); err != nil {
		t.Errorf("Bad Error Format")
	}
	return dat["message"], dat["error"]
}

// MergeUserValuesIntoEntityConditionNew does the same as the other, but with the
// "New" conditions.
func MergeUserValuesIntoEntityConditionNew(dsc ds.EntityConditionNew, userInfo auth.UserInfo) (ds.EntityConditionNew, error) {
	if val, ok := userInfo.GetFieldValue(dsc.Value); ok {
		dsc.Value = val
		dsc.Type = "fieldvalue"
		return dsc, nil
	}
	return ds.EntityConditionNew{}, fmt.Errorf("User Field (named %v) in User-Based Condition (named %v) is invalid", dsc.Value, dsc.Name)
}

// ConditionTransformer processes a condition based on user info
type ConditionTransformer func(ds.EntityConditionNew) (map[string]interface{}, error)

// ConditionBuilder returns a function that will check permission and process a condition
func ConditionBuilder(userInfo auth.UserInfo) ConditionTransformer {
	return func(dsc ds.EntityConditionNew) (map[string]interface{}, error) {
		// Loop over dso conditions
		if dsc.AlwaysOn && dsc.ExecuteOnQuery {
			var err error
			// If they condition is based on user info, take the userInfo
			// from Pliny and map the value from userInfo into the condition
			if dsc.Type == "userinfo" {
				dsc, err = MergeUserValuesIntoEntityConditionNew(dsc, userInfo)
				if err != nil {
					return nil, err
				}
			}

			return map[string]interface{}{
				"field": dsc.Field,
				"name":  dsc.Name,
				"type":  dsc.Type,
				"value": dsc.Value,
			}, nil
		}

		return nil, nil
	}
}
