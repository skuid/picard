//go:generate swagger generate spec -m -o ./swagger.json
// Package classification Warden API.
//
// Warden provides a security layer for registering data sources and the objects
// in those data sources (entities, actions, fields, and permissions)
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     Host: localhost
//     BasePath: /api/v2
//     Version: 0.0.1
//     License: MIT http://opensource.org/licenses/MIT
//     Contact: John Doe<john.doe@example.com> http://john.doe.com
//
//     Consumes:
//     - application/json
//     - application/vnd.api+json
//
//     Produces:
//     - application/json
//     - application/vnd.api+json
//
//     Extensions:
//     x-skuid-session-id: value
//
// swagger:meta
package main

import (
	"os"

	"go.uber.org/zap"

	"github.com/skuid/spec"
	"github.com/skuid/warden/cmd"
)

func main() {
	logger, _ := spec.NewStandardLogger()
	zap.ReplaceGlobals(logger)
	if err := cmd.RootCmd.Execute(); err != nil {
		logger.Error("Encountered an error with root cobra command", zap.Error(err))
		os.Exit(-1)
	}

}
