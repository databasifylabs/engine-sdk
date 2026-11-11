package command

import (
	"context"
	"errors"
	"testing"

	"github.com/databasifylabs/engine-sdk/contract"
	"github.com/databasifylabs/engine-sdk/ctl"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
)

type fakeSupervisor struct {
	active  map[string]bool
	started []ctl.Transient
	stopped []string
}

func (f *fakeSupervisor) Active(unit string) bool { return f.active[unit] }

func (f *fakeSupervisor) Run(transient ctl.Transient) error {
	f.started = append(f.started, transient)
	f.active[transient.Unit] = true

	return nil
}

func (f *fakeSupervisor) Do(verb string, units ...string) error {
	f.stopped = append(f.stopped, units...)

	return nil
}

func testOperations(t *testing.T) (operations, *fakeSupervisor) {
	t.Helper()
	supervisor := &fakeSupervisor{active: map[string]bool{}}

	return operations{dir: t.TempDir(), units: supervisor, executable: "/usr/local/bin/databasify"}, supervisor
}

func readyRoute(err error) route {
	return long(nodes, func(r requests.NodeReady) string { return r.Key }, func(Engine) func(context.Context, requests.NodeReady, contract.Report) error {
		return func(_ context.Context, _ requests.NodeReady, report contract.Report) error {
			report("waiting")

			return err
		}
	})
}

const readyInput = `{"key":"ready-1","expectPrimary":true}`

func TestStartSavesARunningOperationAndRunsItsUnit(t *testing.T) {
	o, supervisor := testOperations(t)
	operation, err := o.start("node ready", readyRoute(nil), []byte(readyInput))
	if err != nil {
		t.Fatal(err)
	}
	if operation.State != enums.OperationStateRunning || operation.Key != "ready-1" {
		t.Fatalf("operation = %+v", operation)
	}
	if len(supervisor.started) != 1 || supervisor.started[0].Unit != "databasify-op-ready-1" || supervisor.started[0].Argv[1] != "operation-run" {
		t.Fatalf("started = %+v", supervisor.started)
	}
}

func TestStartingAnExistingKeyReturnsItWithoutASecondUnit(t *testing.T) {
	o, supervisor := testOperations(t)
	_, _ = o.start("node ready", readyRoute(nil), []byte(readyInput))
	if _, err := o.start("node ready", readyRoute(nil), []byte(readyInput)); err != nil {
		t.Fatal(err)
	}
	if len(supervisor.started) != 1 {
		t.Fatalf("started %d units", len(supervisor.started))
	}
}

func TestAKeyUsedByAnotherKindIsBusy(t *testing.T) {
	o, _ := testOperations(t)
	_, _ = o.start("node ready", readyRoute(nil), []byte(readyInput))
	_, err := o.start("cluster enable", readyRoute(nil), []byte(readyInput))
	var failure Failure
	if !errors.As(err, &failure) || failure.Code != enums.ErrorCodeBusy {
		t.Fatalf("err = %v", err)
	}
}

func TestARunningOperationWhoseUnitVanishedIsInterrupted(t *testing.T) {
	o, supervisor := testOperations(t)
	_, _ = o.start("node ready", readyRoute(nil), []byte(readyInput))
	supervisor.active = map[string]bool{}
	value, err := o.status("ready-1")
	if err != nil || value.State != enums.OperationStateInterrupted {
		t.Fatalf("value = %+v, err = %v", value, err)
	}
}

func TestCancelStopsTheUnit(t *testing.T) {
	o, supervisor := testOperations(t)
	_, _ = o.start("node ready", readyRoute(nil), []byte(readyInput))
	value, err := o.cancel("ready-1")
	if err != nil || value.State != enums.OperationStateCancelled || supervisor.stopped[0] != "databasify-op-ready-1" {
		t.Fatalf("value = %+v, err = %v, stopped = %v", value, err, supervisor.stopped)
	}
}

func TestRunRecordsSuccessAndFailure(t *testing.T) {
	for name, runErr := range map[string]error{"succeeded": nil, "failed": errors.New("clone broke")} {
		t.Run(name, func(t *testing.T) {
			o, _ := testOperations(t)
			ready := readyRoute(runErr)
			_, _ = o.start("node ready", ready, []byte(readyInput))
			_ = o.runNow(context.Background(), nil, ready, "ready-1")
			r, _, _ := o.load("ready-1")
			if string(r.Operation.State) != name || r.Operation.Phase != "waiting" || r.Operation.FinishedAt == nil {
				t.Fatalf("operation = %+v", r.Operation)
			}
			if runErr != nil && r.Operation.Error.Message != "clone broke" {
				t.Fatalf("error = %+v", r.Operation.Error)
			}
		})
	}
}

func TestUnknownStatusKeyIsNotFound(t *testing.T) {
	o, _ := testOperations(t)
	_, err := o.status("missing")
	var failure Failure
	if !errors.As(err, &failure) || failure.Code != enums.ErrorCodeNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestAFailedOperationKeepsItsErrorCode(t *testing.T) {
	o, _ := testOperations(t)
	ready := readyRoute(Invalid("add a primary key"))
	_, _ = o.start("node ready", ready, []byte(readyInput))
	_ = o.runNow(context.Background(), nil, ready, "ready-1")

	r, _, _ := o.load("ready-1")

	if r.Operation.Error == nil || r.Operation.Error.Code != enums.ErrorCodeInvalidRequest {
		t.Fatalf("error = %+v", r.Operation.Error)
	}
}
