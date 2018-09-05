package entity

import (
	"errors"
)

func parseEntity(vars map[string]string) (string, error) {
	entity, found := vars["entity"]

	if !found {
		return "", errors.New("Entity name not provided")
	}

	return entity, nil
}

func makeEntityBody(entity string) map[string]interface{} {
	body := make(map[string]interface{})
	body["entity"] = entity
	return body
}
