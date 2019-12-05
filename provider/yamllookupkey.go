package provider

import (
	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

var YamlDataKey = `yaml::data`

// YamlLookupKey is a LookupKey function that uses the YamlData DataHash function to find the data and caches the result.
// It is mainly intended for testing purposes but can also be used as a complete replacement of a Configured hiera
// setup.
func YamlLookupKey(pc hiera.ProviderContext, key string) dgo.Value {
	sc, ok := pc.(hieraapi.ServerContext)
	if !ok {
		return nil
	}
	data, ok := sc.CachedValue(YamlDataKey)
	if !ok {
		iv := sc.Invocation()
		data = YamlData(iv.ServerContext(nil, vf.Map(`path`, iv.SessionOptions().Get(`path`))))
		sc.Cache(YamlDataKey, data)
	}
	hash, _ := data.(dgo.Map)
	return hash.Get(key)
}
