package host

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databasifylabs/engine-sdk/requests"
)

func homeOf(t *testing.T, srv string, engine string, databaseId string) string {
	t.Helper()
	home := filepath.Join(srv, engine)
	if err := os.MkdirAll(filepath.Join(home, "data"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, databaseMarker), []byte(databaseId), 0o644); err != nil {
		t.Fatal(err)
	}

	return home
}

func TestRemoveStopsAndKeepsData(t *testing.T) {
	srv, calls := t.TempDir(), []string{}
	home := homeOf(t, srv, "postgres", "db1")

	if err := remove(fakeDocker{calls: &calls}, srv, func(string) error { return nil }, requests.HostRemove{DatabaseId: "db1"}); err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(calls, "stop --time 960 databasify-postgres") {
		t.Errorf("calls = %v", calls)
	}
	if _, err := os.Stat(home); err != nil {
		t.Errorf("the data was removed: %v", err)
	}
}

func TestRemoveEraseDeletesOnlyTheMatchingDirectory(t *testing.T) {
	srv, calls, restored := t.TempDir(), []string{}, false
	mysql := homeOf(t, srv, "mysql", "db1")
	postgres := homeOf(t, srv, "postgres", "db2")

	err := remove(fakeDocker{calls: &calls}, srv, func(engine string) error { restored = engine == "mysql"; return nil }, requests.HostRemove{DatabaseId: "db1", Erase: true})

	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(calls, "rm -v -f databasify-mysql") || !restored {
		t.Errorf("calls = %v, restored = %v", calls, restored)
	}
	if _, err := os.Stat(mysql); !os.IsNotExist(err) {
		t.Errorf("%s still exists", mysql)
	}
	if _, err := os.Stat(postgres); err != nil {
		t.Errorf("%s was touched: %v", postgres, err)
	}
}

func TestRemoveWithoutAMatchingDirectoryChangesNothing(t *testing.T) {
	srv, calls := t.TempDir(), []string{}
	postgres := homeOf(t, srv, "postgres", "db2")

	if err := remove(fakeDocker{calls: &calls}, srv, func(string) error { t.Fatal("restore ran"); return nil }, requests.HostRemove{DatabaseId: "nope", Erase: true}); err != nil {
		t.Fatal(err)
	}

	if len(calls) != 0 {
		t.Errorf("docker ran: %v", calls)
	}
	if _, err := os.Stat(postgres); err != nil {
		t.Errorf("%s was touched", postgres)
	}
}
