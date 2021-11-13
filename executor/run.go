package executor

import (
	"context"
	"io"

	"github.com/drone/runner-go/pipeline/runtime"

	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
)

func executeRunStep(ctx context.Context, engine *engine.Engine, r *api.StartStepRequest, out io.Writer) (
	*runtime.State, error) {
	step := toStep(r)
	step.Command = r.Run.Command
	step.Entrypoint = r.Run.Entrypoint

	return engine.Run(ctx, step, out)
}
