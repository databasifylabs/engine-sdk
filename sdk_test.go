package sdk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/contract"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/responses"
)

type primaryNodes struct {
	contract.Nodes
}

func (primaryNodes) Status(context.Context) (responses.NodeStatus, error) {
	return responses.NodeStatus{Role: enums.NodeRolePrimary}, nil
}

type fakeEngine struct {
	nodes contract.Nodes
}

func (fakeEngine) Name() string { return "fake" }

func (fakeEngine) Capabilities() contract.Capabilities { return contract.Capabilities{} }

func (e fakeEngine) Nodes() contract.Nodes { return e.nodes }

func (fakeEngine) Accounts() contract.Accounts { return contract.UnsupportedAccounts{} }

func (fakeEngine) Cluster() contract.Cluster { return contract.UnsupportedCluster{} }

func (fakeEngine) Archive() contract.Archive { return contract.UnsupportedArchive{} }

func (fakeEngine) Migration() contract.Migration { return contract.UnsupportedMigration{} }

func (fakeEngine) Pools() contract.Pools { return contract.UnsupportedPools{} }

func (fakeEngine) Config() contract.Config { return contract.UnsupportedConfig{} }

func (fakeEngine) HostProfile() contract.HostProfile { return contract.HostProfile{} }

func runInContainer(t *testing.T, engine contract.Engine, input string, args ...string) (string, int) {
	t.Helper()
	t.Setenv("DATABASIFY_ENGINE", "fake")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stdin"), []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	stdin, err := os.Open(filepath.Join(dir, "stdin"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := os.Create(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}
	args0, stdin0, stdout0 := os.Args, os.Stdin, os.Stdout
	os.Args, os.Stdin, os.Stdout = append([]string{"databasify"}, args...), stdin, stdout
	exit := Run(engine)
	os.Args, os.Stdin, os.Stdout = args0, stdin0, stdout0
	_ = stdin.Close()
	_ = stdout.Close()
	written, err := os.ReadFile(filepath.Join(dir, "stdout"))
	if err != nil {
		t.Fatal(err)
	}

	return string(written), exit
}

func TestRunDispatchesToTheGivenEngine(t *testing.T) {
	stdout, exit := runInContainer(t, fakeEngine{nodes: primaryNodes{contract.UnsupportedNodes{}}}, "{}", "node", "status")
	if exit != 0 || !strings.HasPrefix(stdout, `{"role":"primary",`) {
		t.Fatalf("node status: exit = %d, stdout = %s", exit, stdout)
	}

	stdout, exit = runInContainer(t, fakeEngine{nodes: contract.UnsupportedNodes{}}, "{}", "user", "create")
	if exit != 1 || !strings.Contains(stdout, `"code":"unsupported"`) {
		t.Fatalf("user create: exit = %d, stdout = %s", exit, stdout)
	}
}

func TestEngineInfoReportsTheSdkVersion(t *testing.T) {
	stdout, exit := runInContainer(t, fakeEngine{nodes: contract.UnsupportedNodes{}}, "{}", "engine", "info")

	if exit != 0 || !strings.Contains(stdout, `"sdk":"`+Version+`"`) || strings.Contains(stdout, `"wrapper"`) {
		t.Fatalf("exit = %d, stdout = %s", exit, stdout)
	}
}

func TestRunAnswersUnsupportedForAnUnknownCommand(t *testing.T) {
	stdout, exit := runInContainer(t, fakeEngine{nodes: contract.UnsupportedNodes{}}, "{}", "widget", "frobnicate")

	if exit != 1 || !strings.Contains(stdout, `"code":"unsupported"`) {
		t.Fatalf("exit = %d, stdout = %s", exit, stdout)
	}
}

func TestRunAnswersUsageForAMalformedCommand(t *testing.T) {
	stdout, exit := runInContainer(t, fakeEngine{nodes: contract.UnsupportedNodes{}}, "{}", "Widget", "frobnicate")

	if exit != 64 || stdout != "" {
		t.Fatalf("exit = %d, stdout = %s", exit, stdout)
	}
}

type statusOnlyNodes struct {
	primaryNodes
}

func (statusOnlyNodes) Implements() []string { return []string{"node status"} }

func TestAPartialGroupOffersOnlyTheCommandsItImplements(t *testing.T) {
	engine := fakeEngine{nodes: statusOnlyNodes{primaryNodes{contract.UnsupportedNodes{}}}}

	stdout, exit := runInContainer(t, engine, "{}", "engine", "capabilities")
	if exit != 0 || !strings.Contains(stdout, `"node status"`) || strings.Contains(stdout, `"node start"`) {
		t.Fatalf("engine capabilities: exit = %d, stdout = %s", exit, stdout)
	}
	if stdout, exit := runInContainer(t, engine, "{}", "node", "status"); exit != 0 || !strings.HasPrefix(stdout, `{"role":"primary",`) {
		t.Fatalf("node status: exit = %d, stdout = %s", exit, stdout)
	}
	if stdout, exit := runInContainer(t, engine, "{}", "node", "start"); exit != 1 || !strings.Contains(stdout, `"unsupported"`) {
		t.Fatalf("node start: exit = %d, stdout = %s", exit, stdout)
	}
}

func TestAnEngineWithoutTuningAnswersAnEmptyHostProfile(t *testing.T) {
	stdout, exit := runInContainer(t, fakeEngine{nodes: primaryNodes{contract.UnsupportedNodes{}}}, "{}", "engine", "host-profile")
	if exit != 0 || stdout != "{\"settings\":{},\"floors\":{},\"disableTransparentHugepages\":false}\n" {
		t.Fatalf("engine host-profile: exit = %d, stdout = %q", exit, stdout)
	}
}
