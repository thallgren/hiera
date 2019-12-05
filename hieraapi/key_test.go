package hieraapi_test

import (
	"fmt"
	"testing"

	require "github.com/lyraproj/dgo/dgo_test"
	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hieraapi"
)

func ExampleNewKey_simple() {
	key := hieraapi.NewKey(`simple`)
	fmt.Printf(`%s, %d`, key.Source(), len(key.Parts()))
	// Output: simple, 1
}

func ExampleNewKey_dotted() {
	key := hieraapi.NewKey(`a.b.c`)
	fmt.Printf(`%s, %d`, key.Source(), len(key.Parts()))
	// Output: a.b.c, 3
}

func ExampleNewKey_dotted_int() {
	key := hieraapi.NewKey(`a.3`)
	fmt.Printf(`%T`, key.Parts()[1])
	// Output: int
}

func ExampleNewKey_quoted() {
	key := hieraapi.NewKey(`'a.b.c'`)
	fmt.Printf(`%s, %d`, key.Source(), len(key.Parts()))
	// Output: 'a.b.c', 1
}

func ExampleNewKey_doubleQuoted() {
	key := hieraapi.NewKey(`"a.b.c"`)
	fmt.Printf(`%s, %d`, key.Source(), len(key.Parts()))
	// Output: "a.b.c", 1
}

func ExampleNewKey_quotedDot() {
	key := hieraapi.NewKey(`a.'b.c'`)
	fmt.Printf(`%s, %d, %s`, key.Source(), len(key.Parts()), key.Parts()[1])
	// Output: a.'b.c', 2, b.c
}

func TestNewKey_quotedDotX(t *testing.T) {
	key := hieraapi.NewKey(`a.'b.c'.d`)
	require.Equal(t, 3, len(key.Parts()))
	require.Equal(t, `b.c`, key.Parts()[1])
}

func TestNewKey_quotedQuote(t *testing.T) {
	key := hieraapi.NewKey(`a.b.'c"d"e'`)
	require.Equal(t, `c"d"e`, key.Parts()[2])
}

func TestNewKey_doubleQuotedQuote(t *testing.T) {
	key := hieraapi.NewKey(`a.b."c'd'e"`)
	require.Equal(t, `c'd'e`, key.Parts()[2])
}

func TestNewKey_unterminatedQuoted(t *testing.T) {
	require.Panic(t, func() { hieraapi.NewKey(`a.b."c`) }, `unterminated quote in key 'a\.b\."c'`)
}

func TestNewKey_empty(t *testing.T) {
	require.Panic(t, func() { hieraapi.NewKey(``) }, `key '' contains an empty segment`)
}

func TestNewKey_emptySegment(t *testing.T) {
	require.Panic(t, func() { hieraapi.NewKey(`a..b`) }, `key 'a\.\.b' contains an empty segment`)
}

func TestNewKey_emptySegmentStart(t *testing.T) {
	require.Panic(t, func() { hieraapi.NewKey(`a.`) }, `key 'a\.' contains an empty segment`)
}

func TestNewKey_emptySegmentEnd(t *testing.T) {
	require.Panic(t, func() { hieraapi.NewKey(`.b`) }, `key '\.b' contains an empty segment`)
}

func TestNewKey_firstSegmentIndex(t *testing.T) {
	require.Panic(t, func() { hieraapi.NewKey(`1.a`) }, `key '1\.a' first segment cannot be an index`)
}

func TestKey_Bury_dotted(t *testing.T) {
	v := hieraapi.NewKey(`a.b.c`).Bury(vf.String(`x`))
	require.Equal(t, vf.Map(`b`, vf.Map(`c`, `x`)), v)
}

func TestKey_Bury_dotted_int(t *testing.T) {
	v := hieraapi.NewKey(`a.3`).Bury(vf.String(`x`))
	require.Equal(t, vf.Map(3, `x`), v)
}

func TestKey_Bury_untouched(t *testing.T) {
	v := hieraapi.NewKey(`a`).Bury(vf.String(`x`))
	require.Equal(t, `x`, v)
}
