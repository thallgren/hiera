package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/lyraproj/hiera/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookup_defaultInt(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `23`, `--type`, `int`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "23\n", string(result))
}

func TestLookup_defaultString(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `23`, `--type`, `string`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "\"23\"\n", string(result))
}

func TestLookup_defaultEmptyString(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, ``, `foo`)
	require.NoError(t, err)
	require.Equal(t, "\"\"\n", string(result))
}

func TestLookup_defaultHash(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `{ x: "a", y: 9 }`, `--type`, `map[string](string|int)`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "x: a\ny: 9\n", string(result))
}

func TestLookup_defaultHash_json(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `{ x: "a", y: 9 }`, `--type`, `map[string](string|int)`, `--render-as`, `json`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "{\"x\":\"a\",\"y\":9}\n", string(result))
}

func TestLookup_defaultString_s(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `xyz`, `--render-as`, `s`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "xyz\n", string(result))
}

func TestLookup_defaultString_binary(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `YWJjMTIzIT8kKiYoKSctPUB+`, `--render-as`, `binary`, `foo`)
	require.NoError(t, err)
	require.Equal(t, "abc123!?$*&()'-=@~", string(result))
}

func TestLookup_defaultArray_binary(t *testing.T) {
	result, err := cli.ExecuteLookup(`--default`, `{12, 28, 37, 15}`, `--type`, `[]int`, `--render-as`, `binary`, `foo`)
	require.NoError(t, err)
	require.Equal(t, []byte{12, 28, 37, 15}, result)
}

func TestLookup_facts(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--facts`, `facts.yaml`, `interpolate_a`)
		require.NoError(t, err)
		require.Equal(t, "This is value of a\n", string(result))
	})
}

func TestLookup_fact_interpolated_config(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--facts`, `facts.yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Equal(t, "This is value of c.a\n", string(result))
	})
}

func TestLookup_vars_interpolated_config(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--vars`, `facts.yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Equal(t, "This is value of c.a\n", string(result))
	})
}

func TestLookup_var_interpolated_config(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--var`, `c={a:"the option value"}`, `--var`, `data_file: by_fact`, `interpolate_ca`)
		require.NoError(t, err)
		require.Equal(t, "This is the option value\n", string(result))
	})
}

func TestLookup_fact_directly(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--facts`, `facts.yaml`, `--config`, `fact_directly.yaml`, `the_fact`)
		require.NoError(t, err)
		require.Equal(t, "value of the_fact\n", string(result))
	})
}

func TestLookup_nullentry(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`nullentry`)
		require.NoError(t, err)
		require.Equal(t, "nv: null\n", string(result))
	})
}

func TestLookup_emptyMap(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--config`, `empty_map.yaml`, `--render-as`, `json`, `empty_map`)
		require.NoError(t, err)
		require.Equal(t, "{}\n", string(result))
	})
}

func TestLookup_emptySubMap(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--config`, `empty_map.yaml`, `--render-as`, `json`, `empty_sub_map`)
		require.NoError(t, err)
		require.Equal(t, "{\"x\":\"the x\",\"empty\":{}}\n", string(result))
	})
}

func TestLookup_emptySubMapInArray(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--config`, `empty_map.yaml`, `--render-as`, `json`, `empty_sub_map_in_array`)
		require.NoError(t, err)
		require.Equal(t, "[{}]\n", string(result))
	})
}

func TestLookup_explain(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--explain`, `--facts`, `facts.yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Regexp(t,
			`\ASearching for "interpolate_ca"
  Merge strategy "first found strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        No such key: "interpolate_ca"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}.yaml"
        Interpolation on "This is %\{c\.a\}"
          Sub key: "a"
            Found key: "a" value: "value of c.a"
        Found key: "interpolate_ca" value: "This is value of c\.a"
    Merged result: "This is value of c\.a"
\z`, string(result))
	})
}

func TestLookup_explain_yaml(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--explain`, `--facts`, `facts.yaml`, `--render-as`, `yaml`, `interpolate_ca`)
		require.NoError(t, err)
		require.Regexp(t,
			`\A__type: hiera\.explainer
branches:
  - __type: hiera\.explainLookup
    branches:
      - __type: hiera\.explainMerge
        branches:
          - __type: hiera\.explainDataProvider
            branches:
              - __type: hiera\.explainLocation
                event: 5
                key: interpolate_ca
                location:
                    __type: hiera\.path
                    original: common\.yaml
                    resolved: .*/testdata/hiera/common\.yaml
                    exists: true
            providerName: data_hash function 'yaml_data'
          - __type: hiera\.explainDataProvider
            branches:
              - __type: hiera\.explainLocation
                branches:
                  - __type: hiera\.explainInterpolate
                    branches:
                      - __type: hiera\.explainSubLookup
                        branches:
                          - __type: hiera\.explainKeySegment
                            event: 1
                            key: a
                            value: value of c\.a
                            segment: a
                        subKey: c\.a
                    expression: This is %\{c\.a\}
                event: 1
                key: interpolate_ca
                value: This is value of c\.a
                location:
                    __type: hiera\.path
                    original: named_%\{data_file\}\.yaml
                    resolved: .*/testdata/hiera/named_by_fact\.yaml
                    exists: true
            providerName: data_hash function 'yaml_data'
        event: 6
        value: This is value of c\.a
    key: interpolate_ca
\z`, string(result))
	})
}

func TestLookup_explain_options(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--explain-options`, `--facts`, `facts.yaml`, `hash`)
		require.NoError(t, err)
		require.Regexp(t,
			`\ASearching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        Found key: "lookup_options" value: \{
          "hash": \{
            "merge": "deep"
          \},
          "sense": \{
            "convert_to": "Sensitive"
          \}
        \}
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}\.yaml"
        No such key: "lookup_options"
    Merged result: \{
      "hash": \{
        "merge": "deep"
      \},
      "sense": \{
        "convert_to": "Sensitive"
      \}
    \}
Using merge options from "lookup_options" hash
\z`, string(result))
	})
}

func TestLookup_explain_explain_options(t *testing.T) {
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--explain`, `--explain-options`, `--facts`, `facts.yaml`, `hash`)
		require.NoError(t, err)
		require.Regexp(t,
			`\ASearching for "lookup_options"
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        Found key: "lookup_options" value: \{
          "hash": \{
            "merge": "deep"
          \},
          "sense": \{
            "convert_to": "Sensitive"
          \}
        \}
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}\.yaml"
        No such key: "lookup_options"
    Merged result: \{
      "hash": \{
        "merge": "deep"
      \},
      "sense": \{
        "convert_to": "Sensitive"
      \}
    \}
Searching for "hash"
  Using merge options from "lookup_options" hash
  Merge strategy "deep merge strategy"
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/common\.yaml"
        Original path: "common\.yaml"
        Found key: "hash" value: \{
          "one": 1,
          "two": "two",
          "three": \{
            "a": "A",
            "c": "C"
          \}
        \}
    data_hash function 'yaml_data'
      Path "[^"]*/testdata/hiera/named_by_fact\.yaml"
        Original path: "named_%\{data_file\}\.yaml"
        Found key: "hash" value: \{
          "one": "overwritten one",
          "three": \{
            "a": "overwritten A",
            "b": "B",
            "c": "overwritten C"
          \}
        \}
    Merged result: \{
      "one": 1,
      "two": "two",
      "three": \{
        "a": "A",
        "c": "C",
        "b": "B"
      \}
    \}
\z`, string(result))
	})
}

func TestLookupKey_plugin(t *testing.T) {
	ensureTestPlugin(t)
	inTestdata(func() {
		result, err := cli.ExecuteLookup(`--config`, `lookup_key_plugin.yaml`, `a`)
		require.NoError(t, err)
		require.Equal(t, "option a\n", string(result))
	})
}

func TestDataHash_refuseToDie(t *testing.T) {
	ensureTestPlugin(t)
	inTestdata(func() {
		_, err := cli.ExecuteLookup(`--config`, `refuse_to_die_plugin.yaml`, `a`)
		if assert.Error(t, err) {
			require.Regexp(t, `net/http: request canceled`, err.Error())
		}
	})
}

func TestDataHash_panic(t *testing.T) {
	ensureTestPlugin(t)
	inTestdata(func() {
		_, err := cli.ExecuteLookup(`--config`, `panic_plugin.yaml`, `a`)
		if assert.Error(t, err) {
			require.Regexp(t, `500 Internal Server Error: dit dit dit daah daah daah dit dit dit`, err.Error())
		}
	})
}

var once = sync.Once{}

func ensureTestPlugin(t *testing.T) {
	once.Do(func() {
		t.Helper()
		cw, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}

		if err = os.Chdir(filepath.Join(`testdata`, `hieratestplugin`)); err != nil {
			t.Fatal(err)
		}

		defer func() {
			_ = os.Chdir(cw)
		}()

		pe := `hieratestplugin`
		ps := pe + `.go`
		if runtime.GOOS == `windows` {
			pe += `.exe`
		}

		cmd := exec.Command(`go`, `build`, `-o`, filepath.Join(`..`, `plugin`, pe), ps)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err = cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
}

func inTestdata(f func()) {
	cw, err := os.Getwd()
	if err == nil {
		err = os.Chdir(`testdata`)
		if err == nil {
			defer func() {
				_ = os.Chdir(cw)
			}()
			f()
		}
	}
	if err != nil {
		panic(err)
	}
}
