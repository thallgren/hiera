package internal

import (
	"fmt"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hierasdk/hiera"
)

type DataDigProvider struct {
	hierarchyEntry hieraapi.Entry
	providerFunc   hiera.DataDig
}

func (dh *DataDigProvider) Hierarchy() hieraapi.Entry {
	return dh.hierarchyEntry
}

func (dh *DataDigProvider) LookupKey(key hieraapi.Key, ic hieraapi.Invocation, location hieraapi.Location) dgo.Value {
	opts := dh.hierarchyEntry.Options()
	if location != nil {
		opts = optionsWithLocation(opts, location.Resolved())
	}
	value := dh.providerFunction(ic)(ic.ServerContext(dh.hierarchyEntry, opts), vf.Values(key.Parts()...))
	if value != nil {
		ic.ReportFound(key.Source(), value)
		value = key.Bury(value)
	} else {
		ic.ReportNotFound(key)
	}
	return value
}

func (dh *DataDigProvider) providerFunction(ic hieraapi.Invocation) (pf hiera.DataDig) {
	if dh.providerFunc == nil {
		dh.providerFunc = dh.loadFunction(ic)
	}
	return dh.providerFunc
}

func (dh *DataDigProvider) loadFunction(ic hieraapi.Invocation) (pf hiera.DataDig) {
	he := dh.hierarchyEntry
	if f, ok := loadFunction(ic, he); ok {
		return func(pc hiera.ProviderContext, key dgo.Array) dgo.Value {
			return f.(dgo.Function).Call(vf.Values(ic, key))[0]
		}
	}
	ic.ReportText(func() string { return fmt.Sprintf(`unresolved function '%s'`, he.Function().Name()) })
	return func(hiera.ProviderContext, dgo.Array) dgo.Value { return nil }
}

func (dh *DataDigProvider) FullName() string {
	return fmt.Sprintf(`data_dig function '%s'`, dh.hierarchyEntry.Function().Name())
}

// NewDataDigProvider creates a new provider with a data_dig function configured from the given entry
func NewDataDigProvider(he hieraapi.Entry) hieraapi.DataProvider {
	return &DataDigProvider{hierarchyEntry: he}
}
