package functions

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/hiera/impl"
)

func init() {
	eval.NewGoFunction(`parse_yaml`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return impl.UnmarshalYaml(c, []byte(args[0].(*types.StringValue).String()))
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Binary`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				return impl.UnmarshalYaml(c, args[0].(*types.BinaryValue).Bytes())
			})
		})
}
