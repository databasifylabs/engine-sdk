package command

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/databasifylabs/engine-sdk/ctl"
	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/units"
)

type record struct {
	Operation dtos.Operation  `json:"operation"`
	Request   json.RawMessage `json:"request"`
}

type supervisor interface {
	Active(unit string) bool
	Run(transient ctl.Transient) error
	Do(verb string, units ...string) error
}

type operations struct {
	dir        string
	units      supervisor
	executable string
}

var operationsRuntime = func() operations {
	paths := layout.FromEnv()
	executable, _ := os.Executable()

	return operations{dir: filepath.Join(paths.Home, "state", "operations"), units: units.Control{Socket: paths.Socket()}, executable: executable}
}

func unitOf(key string) string {
	return "databasify-op-" + key
}

func (o operations) start(kind string, r route, input []byte) (dtos.Operation, error) {
	key, err := r.key(input)
	if err != nil {
		return dtos.Operation{}, err
	}

	return o.begin(kind, key, input)
}

func (o operations) begin(kind string, key string, input []byte) (dtos.Operation, error) {
	if existing, found, err := o.load(key); found || err != nil {
		return o.existing(existing, kind, err)
	}
	rec := record{Operation: dtos.Operation{Key: key, Kind: kind, State: enums.OperationStateRunning, Phase: "starting", StartedAt: now()}, Request: input}
	if err := o.save(rec); err != nil {
		return dtos.Operation{}, err
	}
	transient := ctl.Transient{Unit: unitOf(key), Argv: []string{o.executable, "operation-run", key}, Collect: true}
	if err := o.units.Run(transient); err != nil {
		return dtos.Operation{}, o.finish(rec, err)
	}

	return rec.Operation, nil
}

func (o operations) existing(r record, kind string, err error) (dtos.Operation, error) {
	if err != nil {
		return dtos.Operation{}, err
	}
	if r.Operation.Kind != kind {
		return dtos.Operation{}, Failure{Code: enums.ErrorCodeBusy, Message: "operation " + r.Operation.Key + " is a " + r.Operation.Kind}
	}

	return o.current(r)
}

func (o operations) status(key string) (dtos.Operation, error) {
	r, found, err := o.load(key)
	if err != nil {
		return dtos.Operation{}, err
	}
	if !found {
		return dtos.Operation{}, NotFound("no operation with key " + key)
	}

	return o.current(r)
}

func (o operations) current(r record) (dtos.Operation, error) {
	if r.Operation.State != enums.OperationStateRunning || o.units.Active(unitOf(r.Operation.Key)) {
		return r.Operation, nil
	}
	latest, _, err := o.load(r.Operation.Key)
	if err != nil || latest.Operation.State != enums.OperationStateRunning {
		return latest.Operation, err
	}

	return o.settle(latest, enums.OperationStateInterrupted, errors.New("the operation stopped without finishing"))
}

func (o operations) cancel(key string) (dtos.Operation, error) {
	r, found, err := o.load(key)
	if err != nil {
		return dtos.Operation{}, err
	}
	if !found {
		return dtos.Operation{}, NotFound("no operation with key " + key)
	}
	if r.Operation.State != enums.OperationStateRunning {
		return r.Operation, nil
	}
	if err := o.units.Do("stop", unitOf(key)); err != nil {
		return dtos.Operation{}, err
	}

	return o.settle(r, enums.OperationStateCancelled, nil)
}

func (o operations) runNow(ctx context.Context, engine Engine, r route, key string) error {
	rec, found, err := o.load(key)
	if err != nil || !found {
		return errors.Join(err, errors.New("no operation with key "+key))
	}
	report := func(phase string) {
		rec.Operation.Phase = phase
		_ = o.save(rec)
	}

	err = r.run(ctx, engine, rec.Request, report)

	return o.finish(rec, err)
}

func (o operations) finish(r record, err error) error {
	if latest, found, _ := o.load(r.Operation.Key); found {
		r.Operation.Phase = latest.Operation.Phase
	}
	if err == nil {
		_, saveErr := o.settle(r, enums.OperationStateSucceeded, nil)

		return saveErr
	}
	_, saveErr := o.settle(r, enums.OperationStateFailed, err)

	return errors.Join(err, saveErr)
}

func (o operations) settle(r record, state enums.OperationState, failure error) (dtos.Operation, error) {
	finished := now()
	r.Operation.State = state
	r.Operation.FinishedAt = &finished
	if failure != nil {
		r.Operation.Error = &dtos.Error{Code: errorCodeOf(state, failure), Message: failure.Error(), Details: map[string]string{}}
	}

	return r.Operation, o.save(r)
}

func errorCodeOf(state enums.OperationState, failure error) enums.ErrorCode {
	if state == enums.OperationStateInterrupted {
		return enums.ErrorCodeUnavailable
	}
	var typed Failure
	if errors.As(failure, &typed) {
		return typed.Code
	}

	return enums.ErrorCodeInternal
}

func (o operations) load(key string) (record, bool, error) {
	data, err := os.ReadFile(o.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return record{}, false, nil
	}
	if err != nil {
		return record{}, false, err
	}
	var r record

	return r, true, json.Unmarshal(data, &r)
}

func (o operations) save(r record) error {
	if err := os.MkdirAll(o.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	temporary := o.path(r.Operation.Key) + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}

	return os.Rename(temporary, o.path(r.Operation.Key))
}

func (o operations) path(key string) string {
	return filepath.Join(o.dir, key+".json")
}

func now() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

func RunOperation(ctx context.Context, engine Engine, key string) int {
	o := operationsRuntime()
	r, found, err := o.load(key)
	if err != nil || !found {
		return ctl.ExitInternal
	}
	if r.Operation.Kind == taskKind {
		return exitOf(o.runTask(ctx, key, os.Stdout))
	}
	route, known := routes[r.Operation.Kind]
	if !known || !route.long() {
		_ = o.finish(r, errors.New("this engine cannot run "+r.Operation.Kind))

		return ctl.ExitInternal
	}

	return exitOf(o.runNow(ctx, engine, route, key))
}

func exitOf(err error) int {
	if err != nil {
		return 1
	}

	return 0
}
