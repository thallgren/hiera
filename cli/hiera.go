package cli

import (
	"context"
	"fmt"

	"github.com/lyraproj/dgo/vf"
	"github.com/lyraproj/hiera/hiera"
	"github.com/lyraproj/hiera/hieraapi"
	"github.com/lyraproj/hiera/internal"
	"github.com/lyraproj/hiera/provider"
	sdk "github.com/lyraproj/hierasdk/hiera"
	"github.com/spf13/cobra"
)

var helpTemplate = `Description:
  {{rpad .Long 10}}

Usage:{{if .Runnable}}{{if .HasAvailableFlags}}
  {{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample }}

Examples:
  {{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
{{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}
`

type OptString struct {
	value *string
}

func (s *OptString) Type() string {
	return "stringpointer"
}

func (s *OptString) String() string {
	if s == nil || s.value == nil {
		return ``
	}
	return *s.value
}

func (s *OptString) Set(v string) error {
	s.value = &v
	return nil
}

func (s *OptString) StringPointer() *string {
	return s.value
}

var (
	cmdOpts  hiera.CommandOptions
	dflt     OptString
	logLevel string
	config   string
	facts    []string
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lookup <key> [<key> ...]",
		Short:   `MergeLookup - Perform lookups in Hiera data storage`,
		Long:    "MergeLookup - Perform lookups in Hiera data storage.\n  Find more information at: https://github.com/lyraproj/hiera",
		Version: fmt.Sprintf("%v", getVersion()),
		RunE:    cmdLookup,
		Args:    cobra.MinimumNArgs(1)}

	flags := cmd.Flags()
	flags.StringVar(&logLevel, `loglevel`, `error`, `error/warn/info/debug`)
	flags.StringVar(&cmdOpts.Merge, `merge`, `first`, `first/unique/hash/deep`)
	flags.StringVar(&config, `config`, ``, `path to the hiera config file. Overrides <current directory>/hiera.yaml`)
	flags.Var(&dflt, `default`, `a value to return if Hiera can't find a value in data`)
	flags.StringVar(&cmdOpts.Type, `type`, ``, `assert that the value has the specified type`)
	flags.StringVar(&cmdOpts.RenderAs, `render-as`, ``, `s/json/yaml/binary: Specify the output format of the results; s means plain text`)
	flags.BoolVar(&cmdOpts.ExplainData, `explain`, false, `Explain the details of how the lookup was performed and where the final value came from (or the reason no value was found)`)
	flags.BoolVar(&cmdOpts.ExplainOptions, `explain-options`, false, `Explain whether a lookup_options hash affects this lookup, and how that hash was assembled`)
	flags.StringArrayVar(&cmdOpts.VarPaths, `vars`, nil, `path to a JSON or YAML file that contains key-value mappings to become variables for this lookup`)
	flags.StringArrayVar(&cmdOpts.Variables, `var`, nil, `a key:value or key=value where value is literal expressed using Puppet DSL`)
	flags.StringArrayVar(&facts, `facts`, nil, `alias for --vars for compatibility with Puppet's ruby version of Hiera`)

	cmd.SetHelpTemplate(helpTemplate)
	return cmd
}

func cmdLookup(cmd *cobra.Command, args []string) error {
	cmdOpts.Default = dflt.StringPointer()
	configOptions := vf.MutableMap()
	configOptions.Put(
		provider.LookupKeyFunctions, []sdk.LookupKey{provider.ConfigLookupKey, provider.Environment})

	if config != `` {
		configOptions.Put(hieraapi.HieraConfig, config)
	}
	if len(facts) > 0 {
		cmdOpts.VarPaths = append(cmdOpts.VarPaths, facts...)
	}

	return hiera.TryWithParent(context.Background(), provider.MuxLookupKey, configOptions, func(c hieraapi.Session) error {
		defer internal.KillPlugins()

		hiera.LookupAndRender(c, &cmdOpts, args, cmd.OutOrStdout())
		return nil
	})
}
