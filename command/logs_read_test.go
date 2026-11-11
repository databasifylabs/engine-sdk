package command

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/layout"
	"github.com/databasifylabs/engine-sdk/requests"
)

func logsLayout(t *testing.T, files map[string]string) layout.Layout {
	t.Helper()
	paths := layout.Layout{Home: t.TempDir()}
	if err := os.MkdirAll(paths.Logs(), 0o750); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(paths.Logs(), name), []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
	}

	return paths
}

func unit(name string) *string { return &name }

func TestLogsReadListsUnitsWithoutRotations(t *testing.T) {
	paths := logsLayout(t, map[string]string{"mongod.log": "a\n", "mongod.log.1": "old\n", "pbm-agent.log": "b\n"})

	logs, err := readLogs(paths, requests.LogsRead{Lines: 10})

	if err != nil || !slices.Equal(logs.Units, []string{"mongod", "pbm-agent"}) {
		t.Fatalf("units = %v, err = %v", logs.Units, err)
	}
}

func TestLogsReadReturnsTheLastLines(t *testing.T) {
	paths := logsLayout(t, map[string]string{"mongod.log": "one\ntwo\nthree\n"})

	logs, err := readLogs(paths, requests.LogsRead{Unit: unit("mongod"), Lines: 2})

	if err != nil || !slices.Equal(logs.Lines, []string{"two", "three"}) {
		t.Fatalf("lines = %v, err = %v", logs.Lines, err)
	}
}

func TestLogsReadWithoutAUnitListsUnitsOnly(t *testing.T) {
	paths := logsLayout(t, map[string]string{"mongod.log": "one\n"})

	logs, err := readLogs(paths, requests.LogsRead{Lines: 5})

	if err != nil || logs.Lines == nil || len(logs.Lines) != 0 {
		t.Fatalf("lines = %#v, err = %v", logs.Lines, err)
	}
}

func TestLogsReadUnknownUnitIsNotFound(t *testing.T) {
	paths := logsLayout(t, map[string]string{"mongod.log": "one\n"})

	_, err := readLogs(paths, requests.LogsRead{Unit: unit("nope"), Lines: 5})

	var failure Failure
	if !errors.As(err, &failure) || failure.Code != enums.ErrorCodeNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestLogsReadListsServicesButNotOperations(t *testing.T) {
	paths := logsLayout(t, map[string]string{"mongod.log": "a\n", "databasify.log": "b\n", unitOf("k.1.0") + ".log": "step output\n"})

	listed, err := readLogs(paths, requests.LogsRead{Lines: 5})
	if err != nil || !slices.Equal(listed.Units, []string{"databasify", "mongod"}) {
		t.Fatalf("units = %v, err = %v", listed.Units, err)
	}
	operation, err := readLogs(paths, requests.LogsRead{Unit: unit(unitOf("k.1.0")), Lines: 5})
	if err != nil || !slices.Equal(operation.Lines, []string{"step output"}) {
		t.Fatalf("lines = %v, err = %v", operation.Lines, err)
	}
}
