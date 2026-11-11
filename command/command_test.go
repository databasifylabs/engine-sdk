package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/contract"
	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

type fakeAccounts struct {
	create func(requests.DatabaseCreate) (responses.DatabaseCreate, error)
}

func (f fakeAccounts) List(context.Context) (responses.AccountList, error) {
	return responses.AccountList{}, nil
}

func (f fakeAccounts) CreateDatabase(_ context.Context, request requests.DatabaseCreate) (responses.DatabaseCreate, error) {
	return f.create(request)
}

func (f fakeAccounts) DropDatabase(context.Context, requests.DatabaseDrop) (responses.DatabaseDrop, error) {
	return responses.DatabaseDrop{}, nil
}

func (f fakeAccounts) CreateUser(context.Context, requests.UserCreate) (responses.UserCreate, error) {
	return responses.UserCreate{}, nil
}

func (f fakeAccounts) SetPassword(context.Context, requests.UserSetPassword) error {
	return nil
}

func (f fakeAccounts) Grant(context.Context, requests.UserGrant) error { return nil }

func (f fakeAccounts) DropUser(context.Context, requests.UserDrop) (responses.UserDrop, error) {
	return responses.UserDrop{}, nil
}

type fakeNodes struct{ contract.Nodes }

func (fakeNodes) Reset(context.Context, requests.NodeReset) error { return nil }

func (fakeNodes) Insights(context.Context) (responses.QueryInsights, error) {
	return responses.QueryInsights{}, nil
}

type fakeCluster struct{ contract.Cluster }

func (fakeCluster) Rejoin(context.Context, requests.ClusterRejoin, contract.Report) error {
	return nil
}

func (fakeCluster) UpdateMembership(context.Context, requests.ClusterMembership) error { return nil }

type fakeEngine struct {
	accounts contract.Accounts
	nodes    contract.Nodes
	cluster  contract.Cluster
	profile  contract.HostProfile
}

func (fakeEngine) Name() string { return "fake" }

func (e fakeEngine) HostProfile() contract.HostProfile { return e.profile }

func (fakeEngine) Capabilities() contract.Capabilities {
	pool := 25062

	return contract.Capabilities{Ports: dtos.Ports{Client: 25060, Pool: &pool, Peers: []int{4567}}}
}

func (e fakeEngine) Nodes() contract.Nodes {
	if e.nodes == nil {
		return contract.UnsupportedNodes{}
	}

	return e.nodes
}

func (e fakeEngine) Accounts() contract.Accounts { return e.accounts }

func (e fakeEngine) Cluster() contract.Cluster {
	if e.cluster == nil {
		return contract.UnsupportedCluster{}
	}

	return e.cluster
}

func (fakeEngine) Archive() contract.Archive { return contract.UnsupportedArchive{} }

func (fakeEngine) Migration() contract.Migration { return contract.UnsupportedMigration{} }

func (fakeEngine) Pools() contract.Pools { return contract.UnsupportedPools{} }

func creating(reply responses.DatabaseCreate, err error) fakeEngine {
	return fakeEngine{accounts: fakeAccounts{create: func(requests.DatabaseCreate) (responses.DatabaseCreate, error) { return reply, err }}}
}

func run(engine contract.Engine, stdin string, argv ...string) (int, string) {
	var stdout bytes.Buffer
	code := command.Run(context.Background(), engine, "0.2.0", argv, strings.NewReader(stdin), &stdout)

	return code, stdout.String()
}

func errorCode(t *testing.T, stdout string) string {
	t.Helper()
	var reply responses.Error
	if err := json.Unmarshal([]byte(stdout), &reply); err != nil {
		t.Fatalf("stdout %q: %v", stdout, err)
	}

	return string(reply.Error.Code)
}

func TestRunsTheMethodAndWritesOneReply(t *testing.T) {
	code, stdout := run(creating(responses.DatabaseCreate{Created: true}, nil), `{"name":"shop"}`, "database", "create")
	if code != 0 || stdout != "{\"created\":true}\n" {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}

func TestExitCodesFollowTheErrorCode(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		stdin string
		exit  int
		code  string
	}{
		{"unknown field", nil, `{"name":"shop","x":1}`, 64, "invalid_request"},
		{"failure", command.NotPrimary("replica"), `{"name":"shop"}`, 1, "not_primary"},
		{"plain error", errors.New("boom"), `{"name":"shop"}`, 70, "internal"},
		{"unsupported from method", contract.ErrUnsupported, `{"name":"shop"}`, 1, "unsupported"},
	}
	for _, c := range cases {
		code, stdout := run(creating(responses.DatabaseCreate{}, c.err), c.stdin, "database", "create")
		if code != c.exit || errorCode(t, stdout) != c.code || !strings.Contains(stdout, `"details":{}`) {
			t.Errorf("%s: code = %d, stdout = %q", c.name, code, stdout)
		}
	}
}

func TestPanicExits70WithOneJSONObject(t *testing.T) {
	engine := fakeEngine{accounts: fakeAccounts{create: func(requests.DatabaseCreate) (responses.DatabaseCreate, error) { panic("boom") }}}
	code, stdout := run(engine, `{"name":"shop"}`, "database", "create")
	if code != 70 || errorCode(t, stdout) != "internal" || strings.Count(stdout, "\n") != 1 {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}

func TestAnUnsupportedGroupAnswersUnsupported(t *testing.T) {
	code, stdout := run(creating(responses.DatabaseCreate{}, nil), `{}`, "cluster", "quorum")
	if code != 1 || errorCode(t, stdout) != "unsupported" {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}

func TestEngineInfoReportsTheSdkVersion(t *testing.T) {
	code, stdout := run(creating(responses.DatabaseCreate{}, nil), `{}`, "engine", "info")
	var reply responses.EngineInfo
	if err := json.Unmarshal([]byte(stdout), &reply); code != 0 || err != nil || reply.Engine != "fake" || reply.Sdk != "0.2.0" {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
	if code, _ := run(creating(responses.DatabaseCreate{}, nil), ``, "engine", "info"); code != 64 {
		t.Fatalf("empty stdin: code = %d", code)
	}
}

func TestCapabilitiesListOnlySupportedCommandsSorted(t *testing.T) {
	_, stdout := run(creating(responses.DatabaseCreate{}, nil), `{}`, "engine", "capabilities")
	var reply responses.EngineCapabilities
	_ = json.Unmarshal([]byte(stdout), &reply)
	if !slices.IsSorted(reply.Commands) || !slices.Contains(reply.Commands, "database create") || !slices.Contains(reply.Commands, "engine info") || slices.Contains(reply.Commands, "cluster quorum") {
		t.Fatalf("commands = %v", reply.Commands)
	}
}

func TestCapabilitiesListThePorts(t *testing.T) {
	_, stdout := run(creating(responses.DatabaseCreate{}, nil), `{}`, "engine", "capabilities")
	if !strings.Contains(stdout, `"ports":{"client":25060,"pool":25062,"peers":[4567]}`) {
		t.Fatalf("stdout = %s", stdout)
	}
}

func TestIsWellFormed(t *testing.T) {
	for argv, want := range map[string]bool{"database create": true, "user set-password": true, "widget frobnicate": true, "ctl status": true, "node": false, "node status extra": false, "Node status": false, "node status!": false, "node -status": false} {
		if got := command.IsWellFormed(strings.Fields(argv)); got != want {
			t.Errorf("%s: %v", argv, got)
		}
	}
}

func TestEveryCatalogCommandIsWellFormed(t *testing.T) {
	for _, spec := range contract.Commands {
		if !command.IsWellFormed(strings.Fields(spec.Name)) {
			t.Errorf("%s is not well formed", spec.Name)
		}
	}
}

func TestAnUnknownCommandAnswersUnsupported(t *testing.T) {
	code, stdout := run(fakeEngine{}, ``, "widget", "frobnicate")

	if code != 1 || !strings.Contains(stdout, `"code":"unsupported"`) {
		t.Fatalf("code = %d, stdout = %s", code, stdout)
	}
}

func TestDecodeErrorsNeverEchoPasswords(t *testing.T) {
	for _, stdin := range []string{`{"name":"app","password":"","grants":[]}`, `{"name":"app","password":12345,"grants":[]}`, `{"name":"app","password":"s3cret-Pw","grants":[],"x":1}`} {
		code, stdout := run(creating(responses.DatabaseCreate{}, nil), stdin, "user", "create")
		if code != 64 || strings.Contains(stdout, "s3cret-Pw") || strings.Contains(stdout, "12345") {
			t.Errorf("%s: code = %d, stdout = %q", stdin, code, stdout)
		}
	}
}

func TestNewCommandsRouteToTheirGroups(t *testing.T) {
	engine := fakeEngine{nodes: fakeNodes{contract.UnsupportedNodes{}}, cluster: fakeCluster{contract.UnsupportedCluster{}}}
	cases := map[string]string{
		"node reset":         `{"peers":2,"hosts":["10.0.0.1"]}`,
		"query insights":     `{}`,
		"cluster membership": `{"members":[{"ip":"10.0.0.1","position":1,"voting":true,"cpus":2,"memoryMb":4096,"diskGb":100}]}`,
	}
	for name, stdin := range cases {
		code, stdout := run(engine, stdin, strings.Fields(name)...)
		if code != 0 {
			t.Errorf("%s: code = %d, stdout = %q", name, code, stdout)
		}
	}
}

func TestClusterRejoinRunsAsAnOperation(t *testing.T) {
	spec := contract.Commands[slices.IndexFunc(contract.Commands, func(spec contract.CommandSpec) bool { return spec.Name == "cluster rejoin" })]
	if spec.Kind != enums.KindLong {
		t.Fatalf("cluster rejoin is %s", spec.Kind)
	}
	engine := fakeEngine{nodes: fakeNodes{contract.UnsupportedNodes{}}, cluster: fakeCluster{contract.UnsupportedCluster{}}}
	if code, stdout := run(engine, `{"readOnly":false}`, "cluster", "rejoin"); code != 64 || errorCode(t, stdout) != "invalid_request" {
		t.Fatalf("a rejoin without a key: code = %d, stdout = %q", code, stdout)
	}
}

func (fakeEngine) Config() contract.Config { return contract.UnsupportedConfig{} }

func TestHostProfileOfAnEngineThatTunesNothingIsEmpty(t *testing.T) {
	code, stdout := run(fakeEngine{}, `{}`, "engine", "host-profile")

	if code != 0 || stdout != `{"settings":{},"floors":{},"disableTransparentHugepages":false}`+"\n" {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}

func TestHostProfileRepliesWithTheEngineProfile(t *testing.T) {
	engine := fakeEngine{profile: contract.HostProfile{
		Settings:                    map[string]string{"vm.overcommit_memory": "1"},
		Floors:                      map[string]int64{"net.core.somaxconn": 1024},
		DisableTransparentHugepages: true,
	}}

	code, stdout := run(engine, `{}`, "engine", "host-profile")

	if code != 0 || stdout != `{"settings":{"vm.overcommit_memory":"1"},"floors":{"net.core.somaxconn":1024},"disableTransparentHugepages":true}`+"\n" {
		t.Fatalf("code = %d, stdout = %q", code, stdout)
	}
}
