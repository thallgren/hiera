package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lyraproj/hiera/internal"

	"github.com/lyraproj/dgo/dgo"
	"github.com/lyraproj/dgo/loader"
	"github.com/lyraproj/dgo/streamer"
	"github.com/lyraproj/dgo/tf"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/provider"
	"github.com/lyraproj/hierasdk/hiera"
)

type session struct {
	context.Context
	aliasMap dgo.AliasMap
	dialect  streamer.Dialect
	vars     map[string]interface{}
	scope    dgo.Keyed
	loader   dgo.Loader
}

const hieraCacheKey = `Hiera::Cache`
const hieraTopProviderKey = `Hiera::TopProvider`
const hieraSessionOptionsKey = `Hiera::SessionOptions`
const hieraTopProviderCacheKey = `Hiera::TopProvider::Cache`

// New creates a new Hiera Session which, among other things, holds on to a synchronized
// cache where all loaded things end up.
//
// parent: typically obtained using context.Background() but can be any context.
//
// topProvider: the topmost provider that defines the hierarchy
//
// options: a map[string]any of configuration options
func New(parent context.Context, topProvider hiera.LookupKey, oif interface{}, ldr dgo.Loader) hieraapi.Session {
	if topProvider == nil {
		topProvider = provider.ConfigLookupKey
	}

	options := vf.MutableMap()
	if oif != nil {
		options.PutAll(hieraapi.ToMap(`session options`, oif))
	}

	if options.Get(hieraapi.HieraConfig) == nil {
		var hieraRoot string
		if r := options.Get(hieraapi.HieraRoot); r != nil {
			hieraRoot = r.String()
		} else {
			var err error
			if hieraRoot, err = os.Getwd(); err != nil {
				panic(err)
			}
		}

		var fileName string
		if r := options.Get(hieraapi.HieraConfigFileName); r != nil {
			fileName = r.String()
		} else {
			fileName = `hiera.yaml`
		}
		options.Put(hieraapi.HieraConfig, filepath.Join(hieraRoot, fileName))
	}

	var dialect streamer.Dialect
	if ds, ok := options.Get(hieraapi.HieraDialect).(dgo.String); ok {
		switch ds.String() {
		case "dgo":
			dialect = streamer.DgoDialect()
		case "pcore":
			panic(errors.New(`pcore dialect is not yet implemented`))
		default:
			panic(fmt.Errorf(`unknown dialect '%s'`, ds))
		}
	}
	if dialect == nil {
		dialect = streamer.DgoDialect()
	}

	var scope dgo.Keyed
	if sv, ok := options.Get(hieraapi.HieraScope).(dgo.Keyed); ok {
		// Freeze scope if possible
		if f, ok := sv.(dgo.Freezable); ok {
			sv = f.FrozenCopy().(dgo.Keyed)
		}
		scope = sv
	} else {
		scope = vf.Map()
	}
	options.Freeze()

	vars := map[string]interface{}{
		hieraCacheKey:            &sync.Map{},
		hieraTopProviderKey:      topProvider,
		hieraTopProviderCacheKey: &sync.Map{},
		hieraSessionOptionsKey:   options}

	s := &session{Context: parent, aliasMap: tf.NewAliasMap(), vars: vars, dialect: dialect, scope: scope}
	s.loader = s.newHieraLoader(ldr)
	return s
}

func (s *session) AliasMap() dgo.AliasMap {
	return s.aliasMap
}

func (s *session) Dialect() streamer.Dialect {
	return s.dialect
}

func (s *session) Invocation(si interface{}, explainer hieraapi.Explainer) hieraapi.Invocation {
	var scope dgo.Keyed
	if si == nil {
		scope = s.Scope()
	} else {
		scope = &nestedScope{s.Scope(), hieraapi.ToMap(`invocation scope`, si)}
	}
	return &ivContext{
		Session:    s,
		nameStack:  []string{},
		scope:      scope,
		configPath: s.SessionOptions().Get(hieraapi.HieraConfig).String(),
		explainer:  explainer}
}

func (s *session) Loader() dgo.Loader {
	return s.loader
}

func (s *session) Scope() dgo.Keyed {
	return s.scope
}

func (s *session) Get(key string) interface{} {
	return s.vars[key]
}

func (s *session) TopProvider() hiera.LookupKey {
	if v, ok := s.Get(hieraTopProviderKey).(hiera.LookupKey); ok {
		return v
	}
	panic(notInitialized())
}

func (s *session) TopProviderCache() *sync.Map {
	if v, ok := s.Get(hieraTopProviderCacheKey).(*sync.Map); ok {
		return v
	}
	panic(notInitialized())
}

func (s *session) SessionOptions() dgo.Map {
	if v := s.Get(hieraSessionOptionsKey); v != nil {
		if g, ok := v.(dgo.Map); ok {
			return g
		}
	}
	panic(notInitialized())
}

func notInitialized() error {
	return errors.New(`session is not initialized`)
}

func (s *session) SharedCache() *sync.Map {
	if v, ok := s.Get(hieraCacheKey).(*sync.Map); ok {
		return v
	}
	panic(notInitialized())
}

func (s *session) newHieraLoader(p dgo.Loader) dgo.Loader {
	nsCreator := func(l dgo.Loader, name string) dgo.Loader {
		switch name {
		case `plugin`:
			return internal.CreatePluginLoader(l)
		case `function`:
			return s.createFunctionLoader(l)
		default:
			return nil
		}
	}
	var l dgo.Loader
	if p == nil {
		l = loader.New(nil, ``, nil, nil, nsCreator)
	} else {
		l = p.NewChild(nil, nsCreator)
	}
	return l
}

func (s *session) createFunctionLoader(l dgo.Loader) dgo.Loader {
	m, ok := s.SessionOptions().Get(hieraapi.HieraFunctions).(dgo.Map)
	if !ok {
		m = vf.Map()
	}
	return loader.New(l, `function`, m, nil, nil)
}
