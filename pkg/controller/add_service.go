package controller

import (
	"buildly-service-discovery/pkg/controller/service"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, service.Add)
}
