package impl

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/hiera/lookup"
	"github.com/lyraproj/issue/issue"
	"regexp"
	"strings"
)

var iplPattern = regexp.MustCompile(`%{[^}]*}`)
var emptyInterpolations = map[string]bool {
	``: true,
	`::`: true,
	`""`: true,
	"''": true,
	`"::"`: true,
	"'::'": true,
}

// Interpolate resolves interpolations in the given value and returns the result
func Interpolate(ic lookup.Invocation, value eval.Value, allowMethods bool) eval.Value {
	result, _ := doInterpolate(ic, value, allowMethods)
	return result
}

func doInterpolate(ic lookup.Invocation, value eval.Value, allowMethods bool) (eval.Value, bool) {
	if s, ok := value.(*types.StringValue); ok {
		return interpolateString(ic, s.String(), allowMethods)
	}
	if a, ok := value.(*types.ArrayValue); ok {
		cp := a.AppendTo(make([]eval.Value, 0, a.Len()))
		changed := false
		for i, e := range cp {
			v, c := doInterpolate(ic, e, allowMethods)
			if c {
				changed = true
				cp[i] = v
			}
		}
		if changed {
			a = types.WrapValues(cp)
		}
		return a, changed
	}
	if h, ok := value.(*types.HashValue); ok {
		cp := h.AppendEntriesTo(make([]*types.HashEntry, 0, h.Len()))
		changed := false
		for i, e := range cp {
			k, kc := doInterpolate(ic, e.Key(), allowMethods)
			v, vc := doInterpolate(ic, e.Value(), allowMethods)
			if kc || vc {
				changed = true
				cp[i] = types.WrapHashEntry(k, v)
			}
		}
		if changed {
			h = types.WrapHash(cp)
		}
		return h, changed
	}
	return value, false
}

const scopeMethod = 1
const aliasMethod = 2
const lookupMethod = 3
const literalMethod = 4

var methodMatch = regexp.MustCompile(`^(\w+)\((?:["]([^"]+)["]|[']([^']+)['])\)$`)

func getMethodAndData(expr string, allowMethods bool) (int, string) {
	if groups := methodMatch.FindStringSubmatch(expr); groups != nil {
		if !allowMethods {
			panic(eval.Error(HIERA_INTERPOLATION_METHOD_SYNTAX_NOT_ALLOWED, issue.NO_ARGS))
		}
		data := groups[2]
		if data == `` {
			data = groups[3]
		}
		switch groups[1] {
		case `alias`:
			return aliasMethod, data
		case `hiera`, `lookup`:
			return lookupMethod, data
		case `literal`:
			return literalMethod, data
		case `scope`:
			return scopeMethod, data
		default:
			panic(eval.Error(HIERA_INTERPOLATION_UNKNOWN_INTERPOLATION_METHOD, issue.H{`name`: groups[1]}))
		}
	}
	return scopeMethod, expr
}

func interpolateString(ic lookup.Invocation, str string, allowMethods bool) (result eval.Value, changed bool) {
	changed = false
	if strings.Index(str, `%{`) < 0 {
		result = types.WrapString(str)
		return
	}
	str = iplPattern.ReplaceAllStringFunc(str, func (match string) string {
		expr := strings.TrimSpace(match[2:len(match)-1])
		if emptyInterpolations[expr] {
			return ``
		}
		var methodKey int
		methodKey, expr = getMethodAndData(expr, allowMethods)
		if methodKey == aliasMethod && match != str {
			panic(eval.Error(HIERA_INTERPOLATION_ALIAS_NOT_ENTIRE_STRING, issue.NO_ARGS))
		}

		switch methodKey {
		case literalMethod:
			return expr
		case scopeMethod:
			key := NewKey(expr)
			if val, ok := ic.Scope().Get(key.Root()); ok {
				val, _ = doInterpolate(ic, val, allowMethods)
				if val, ok = key.Dig(val); ok {
					return val.String()
				}
			}
			return ``
		default:
			val := lookup.Lookup(ic, expr, eval.UNDEF, nil)
			if methodKey == aliasMethod {
				result = val
				return ``
			}
			return val.String()
		}
	})
	changed = true
	if result == nil {
		result = types.WrapString(str)
	}
	return

}

func resolveInScope(ic lookup.Invocation, expr string, allowMethods bool) eval.Value {
	key := NewKey(expr)
	if val, ok := ic.Scope().Get(key.Root()); ok {
		val, _ = doInterpolate(ic, val, allowMethods)
		if val, ok = key.Dig(val); ok {
			return val
		}
	}
	return eval.UNDEF
}
