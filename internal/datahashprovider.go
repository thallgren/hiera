package internal

import (
	"fmt"
	"sync"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/hierasdk/hiera"
)

type DataHashProvider struct {
	hierarchyEntry hieraapi.Entry
	providerFunc   hiera.DataHash
	hashes         dgo.Map
	hashesLock     sync.RWMutex
}

func (dh *DataHashProvider) Hierarchy() hieraapi.Entry {
	return dh.hierarchyEntry
}

func (dh *DataHashProvider) LookupKey(key hieraapi.Key, ic hieraapi.Invocation, location hieraapi.Location) dgo.Value {
	root := key.Root()
	if value := dh.dataValue(ic, location, root); value != nil {
		ic.ReportFound(root, value)
		return value
	}
	ic.ReportNotFound(root)
	return nil
}

func (dh *DataHashProvider) dataValue(ic hieraapi.Invocation, location hieraapi.Location, root string) dgo.Value {
	value := dh.dataHash(ic, location).Get(root)
	if value == nil {
		return nil
	}
	return ic.Interpolate(value, true)
}

func (dh *DataHashProvider) providerFunction(ic hieraapi.Invocation) (pf hiera.DataHash) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *DataHashProvider) loadFunction(ic hieraapi.Invocation) hiera.DataHash {
	n := dh.hierarchyEntry.Function().Name()
	switch n {
	case `yaml_data`:
		return provider.YamlData
	case `json_data`:
		return provider.JSONData
	}

	if fn, ok := loadFunction(ic, dh.hierarchyEntry); ok {
		return func(pc hiera.ProviderContext) (value dgo.Map) {
			value = vf.Map()
			v := fn.Call(vf.MutableValues(pc))
			if dv, ok := v[0].(dgo.Map); ok {
				value = dv
			}
			return
		}
	}

	ic.ReportText(func() string { return fmt.Sprintf(`unresolved function '%s'`, n) })
	return func(pc hiera.ProviderContext) dgo.Map {
		return vf.Map()
	}
}

func (dh *DataHashProvider) dataHash(ic hieraapi.Invocation, location hieraapi.Location) (hash dgo.Map) {
	key := ``
	opts := dh.hierarchyEntry.Options()
	if location != nil {
		key = location.Resolved()
		opts = optionsWithLocation(opts, key)
	}

	var ok bool
	dh.hashesLock.RLock()
	hash, ok = dh.hashes.Get(key).(dgo.Map)
	dh.hashesLock.RUnlock()
	if ok {
		return
	}

	dh.hashesLock.Lock()
	defer dh.hashesLock.Unlock()

	if hash, ok = dh.hashes.Get(key).(dgo.Map); ok {
		return hash
	}
	hash = dh.providerFunction(ic)(ic.ServerContext(dh.hierarchyEntry, opts))
	dh.hashes.Put(key, hash)
	return
}

func (dh *DataHashProvider) FullName() string {
	return fmt.Sprintf(`data_hash function '%s'`, dh.hierarchyEntry.Function().Name())
}

// NewDataHashProvider creates a new provider with a data_hash function configured from the given entry
func NewDataHashProvider(he hieraapi.Entry) hieraapi.DataProvider {
	ls := he.Locations()
	return &DataHashProvider{hierarchyEntry: he, hashes: vf.MapWithCapacity(len(ls), nil)}
}

func optionsWithLocation(options dgo.Map, loc string) dgo.Map {
	return options.Merge(vf.Map(`path`, loc))
}
