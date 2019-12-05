package provider

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

// ScopeLookupKey is a function that performs a lookup in the current scope.
func ScopeLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	sc, ok := pc.(hieraapi.ServerContext)
	if !ok {
		return nil
	}
	return sc.Invocation().Scope().Get(key)
}
