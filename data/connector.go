package data

import (
	"fmt"

	"github.com/zorchenhimer/MoviePolls/common"
)

type constructor func(string) (common.DataConnector, error)

var registeredConnectors map[string]constructor

func GetDataConnector(backend, connectionString string) (common.DataConnector, error) {
	dc, ok := registeredConnectors[backend]
	if !ok {
		return nil, fmt.Errorf("Backend %s is not available", backend)
	}

	return dc(connectionString)
}

func register(backend string, initFunc constructor) {
	if registeredConnectors == nil {
		registeredConnectors = map[string]constructor{}
	}

	registeredConnectors[backend] = initFunc
}
