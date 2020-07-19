package controller

import (
	"github.com/uberscott/recert/go/src/operator/pkg/controller/cert"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cert.Add)
}
