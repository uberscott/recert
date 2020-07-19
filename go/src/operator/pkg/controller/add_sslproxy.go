package controller

import (
	"github.com/uberscott/recert/go/src/operator/pkg/controller/sslproxy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, sslproxy.Add)
}
