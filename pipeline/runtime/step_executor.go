package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/harness/lite-engine/api"
	"github.com/harness/lite-engine/engine"
	"github.com/harness/lite-engine/errors"
	"github.com/harness/lite-engine/livelog"
	"github.com/harness/lite-engine/logstream"
	"github.com/harness/lite-engine/pipeline"

	"github.com/drone/runner-go/pipeline/runtime"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

type ExecutionStatus int

type StepStatus struct {
	Status  ExecutionStatus
	State   *runtime.State
	StepErr error
	Outputs map[string]string
}

const (
	NotStarted ExecutionStatus = iota
	Running
	Complete
)

type StepExecutor struct {
	engine     *engine.Engine
	mu         sync.Mutex
	stepStatus map[string]StepStatus
	stepWaitCh map[string][]chan StepStatus
}

func NewStepExecutor(engine *engine.Engine) *StepExecutor {
	return &StepExecutor{
		engine:     engine,
		mu:         sync.Mutex{},
		stepWaitCh: make(map[string][]chan StepStatus),
		stepStatus: make(map[string]StepStatus),
	}
}

func (e *StepExecutor) StartStep(ctx context.Context, r *api.StartStepRequest) error {
	if r.ID == "" {
		return &errors.BadRequestError{Msg: "ID needs to be set"}
	}

	e.mu.Lock()
	_, ok := e.stepStatus[r.ID]
	if ok {
		e.mu.Unlock()
		return nil
	}

	e.stepStatus[r.ID] = StepStatus{Status: Running}
	e.mu.Unlock()

	go func() {
		state, outputs, stepErr := e.executeStep(r)
		status := StepStatus{Status: Complete, State: state, StepErr: stepErr, Outputs: outputs}
		e.mu.Lock()
		e.stepStatus[r.ID] = status
		channels := e.stepWaitCh[r.ID]
		e.mu.Unlock()

		for _, ch := range channels {
			ch <- status
		}
	}()
	return nil
}

func (e *StepExecutor) PollStep(ctx context.Context, r *api.PollStepRequest) (*api.PollStepResponse, error) {
	id := r.ID
	if r.ID == "" {
		return &api.PollStepResponse{}, &errors.BadRequestError{Msg: "ID needs to be set"}
	}

	e.mu.Lock()
	s, ok := e.stepStatus[id]
	if !ok {
		e.mu.Unlock()
		return &api.PollStepResponse{}, &errors.BadRequestError{Msg: "Step has not started"}
	}

	if s.Status == Complete {
		e.mu.Unlock()
		return convertStatus(s), nil
	}

	ch := make(chan StepStatus, 1)
	if _, ok := e.stepWaitCh[id]; !ok {
		e.stepWaitCh[id] = append(e.stepWaitCh[id], ch)
	} else {
		e.stepWaitCh[id] = []chan StepStatus{ch}
	}
	e.mu.Unlock()

	status := <-ch
	return convertStatus(status), nil
}

func (e *StepExecutor) executeStep(r *api.StartStepRequest) (*runtime.State, map[string]string, error) {
	state := pipeline.GetState()
	secrets := append(state.GetSecrets(), r.Secrets...)

	// Create a log stream for step logs
	client := state.GetLogStreamClient()
	wc := livelog.New(client, r.LogKey, getNudges())
	wr := logstream.NewReplacer(wc, secrets)
	go wr.Open() // nolint:errcheck

	// if the step is configured as a daemon, it is detached
	// from the main process and executed separately.
	if r.Detach {
		go func() {
			ctx := context.Background()
			var cancel context.CancelFunc
			if r.Timeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(r.Timeout))
				defer cancel()
			}
			executeRunStep(ctx, e.engine, r, wr) // nolint:errcheck
			wc.Close()
		}()
		return &runtime.State{Exited: false}, nil, nil
	}

	var result error

	ctx := context.Background()
	var cancel context.CancelFunc
	if r.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Second*time.Duration(r.Timeout))
		defer cancel()
	}

	exited, outputs, err := executeRunStep(ctx, e.engine, r, wr)
	if err != nil {
		result = multierror.Append(result, err)
	}

	// close the stream. If the session is a remote session, the
	// full log buffer is uploaded to the remote server.
	if err = wc.Close(); err != nil {
		result = multierror.Append(result, err)
	}

	// if the context was canceled and returns a canceled or
	// DeadlineExceeded error this indicates the step was timed out.
	switch ctx.Err() {
	case context.Canceled, context.DeadlineExceeded:
		return nil, nil, ctx.Err()
	}

	if exited != nil {
		if exited.ExitCode != 0 {
			if wc.Error() != nil {
				result = multierror.Append(result, err)
			}
		}

		if exited.OOMKilled {
			logrus.WithField("id", r.ID).Infoln("received oom kill.")
		} else {
			logrus.WithField("id", r.ID).Infof("received exit code %d\n", exited.ExitCode)
		}
	}
	return exited, outputs, result
}

func convertStatus(status StepStatus) *api.PollStepResponse {
	r := &api.PollStepResponse{
		Exited:  true,
		Outputs: status.Outputs,
	}

	stepErr := status.StepErr

	if status.State != nil {
		r.Exited = status.State.Exited
		r.OOMKilled = status.State.OOMKilled
		r.ExitCode = status.State.ExitCode
		if status.State.OOMKilled {
			stepErr = multierror.Append(stepErr, fmt.Errorf("oom killed"))
		} else if status.State.ExitCode != 0 {
			stepErr = multierror.Append(stepErr, fmt.Errorf("exit status %d", status.State.ExitCode))
		}
	}

	if status.StepErr != nil {
		r.ExitCode = 255
	}

	if stepErr != nil {
		r.Error = stepErr.Error()
	}
	return r
}
