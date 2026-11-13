package host

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
)

const (
	newImage = "ghcr.io/databasifylabs/databasify-postgres:18@sha256:" + "1111111111111111111111111111111111111111111111111111111111111111"
	oldImage = "ghcr.io/databasifylabs/databasify-postgres:18@sha256:" + "2222222222222222222222222222222222222222222222222222222222222222"
)

type fakeDocker struct {
	calls      *[]string
	pulled     bool
	engine     string
	container  string
	withoutSdk bool
}

func (d fakeDocker) Run(args ...string) (string, error) {
	call := strings.Join(args, " ")
	*d.calls = append(*d.calls, call)
	switch {
	case call == "image inspect "+newImage && !d.pulled:
		return "", errors.New("no such image")
	case strings.HasPrefix(call, "image inspect --format") && strings.Contains(call, engineLabelKey):
		return d.engine + "|" + d.sdk(), nil
	case strings.HasPrefix(call, "image inspect --format"):
		return d.engine, nil
	case strings.HasPrefix(call, "container inspect"):
		if d.container == "" {
			return "", errors.New("no such container")
		}

		return d.container, nil
	}

	return "", nil
}

func (d fakeDocker) RunWithInput(input string, args ...string) (string, error) {
	*d.calls = append(*d.calls, strings.Join(args, " "))
	if args[len(args)-1] == "info" {
		return `{"engine":"` + d.engine + `","line":"18","version":"18.1","sdk":"0.2.0"}`, nil
	}

	return "", errors.New("the host asked the image for " + strings.Join(args, " "))
}

func (d fakeDocker) sdk() string {
	if d.withoutSdk {
		return ""
	}

	return "0.2.0"
}

func deployRequest() requests.HostDeploy {
	return requests.HostDeploy{Image: newImage, DatabaseId: "db1", NodeId: "n1"}
}

func deployed(t *testing.T, docker fakeDocker, srv string) []string {
	t.Helper()
	if _, err := deploy(docker, srv, deployRequest()); err != nil {
		t.Fatal(err)
	}

	return *docker.calls
}

func containsCall(calls []string, prefix string) bool {
	return slices.ContainsFunc(calls, func(call string) bool { return strings.HasPrefix(call, prefix) })
}

func TestDeployCreatesTheContainerOnAFreshServer(t *testing.T) {
	srv, calls := t.TempDir(), []string{}
	docker := fakeDocker{calls: &calls, engine: "postgres"}

	reply, err := deploy(docker, srv, deployRequest())

	if err != nil || !reply.Created || reply.Replaced || reply.Engine != "postgres" || reply.Container != "databasify-postgres" {
		t.Fatalf("reply = %+v, err = %v", reply, err)
	}
	if !slices.Contains(calls, "pull "+newImage) {
		t.Errorf("the image was not pulled: %v", calls)
	}
	run := "run -d --name databasify-postgres --network host --ipc host --security-opt label=disable --restart unless-stopped --stop-timeout 960 --ulimit nofile=65536:65536 -v " + srv + "/postgres:" + srv + "/postgres --label io.databasify.database=db1 --label io.databasify.node=n1 -e DATABASIFY_IMAGE=" + newImage + " " + newImage
	if !slices.Contains(calls, run) {
		t.Errorf("no run %q in %v", run, calls)
	}
	if marker, _ := os.ReadFile(filepath.Join(srv, "postgres", ".database")); string(marker) != "db1" {
		t.Errorf(".database = %q", marker)
	}
	for _, dir := range []string{"tls", "conf"} {
		if _, err := os.Stat(filepath.Join(srv, "postgres", dir)); err != nil {
			t.Errorf("%s is missing", dir)
		}
	}
}

func TestDeployLeavesTheSameImageRunning(t *testing.T) {
	calls := []string{}
	docker := fakeDocker{calls: &calls, pulled: true, engine: "postgres", container: newImage + " true db1"}

	reply, err := deploy(docker, t.TempDir(), deployRequest())

	if err != nil || reply.Created || reply.Replaced {
		t.Fatalf("reply = %+v, err = %v", reply, err)
	}
	for _, verb := range []string{"stop", "rm", "run", "start", "pull"} {
		if containsCall(calls, verb+" ") {
			t.Errorf("%s ran: %v", verb, calls)
		}
	}
}

func TestDeployReplacesAContainerOnAnotherImage(t *testing.T) {
	calls := []string{}
	docker := fakeDocker{calls: &calls, pulled: true, engine: "postgres", container: oldImage + " true db1"}

	reply, err := deploy(docker, t.TempDir(), deployRequest())

	if err != nil || !reply.Replaced {
		t.Fatalf("reply = %+v, err = %v", reply, err)
	}
	stop, remove := slices.Index(calls, "stop --time 960 databasify-postgres"), slices.Index(calls, "rm -v databasify-postgres")
	run := slices.IndexFunc(calls, func(call string) bool { return strings.HasPrefix(call, "run -d ") })
	if stop < 0 || remove <= stop || run <= remove {
		t.Fatalf("stop %d, rm %d, run %d in %v", stop, remove, run, calls)
	}
}

func TestDeployStartsAStoppedContainerOnTheSameImage(t *testing.T) {
	calls := deployed(t, fakeDocker{calls: &[]string{}, pulled: true, engine: "postgres", container: newImage + " false db1"}, t.TempDir())

	if !slices.Contains(calls, "start databasify-postgres") || containsCall(calls, "run ") {
		t.Fatalf("calls = %v", calls)
	}
}

func TestDeployRefusesDataFromAnotherDatabase(t *testing.T) {
	srv, calls := t.TempDir(), []string{}
	data := filepath.Join(srv, "postgres", "data")
	if err := os.MkdirAll(data, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "PG_VERSION"), []byte("18"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srv, "postgres", ".database"), []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}
	docker := fakeDocker{calls: &calls, pulled: true, engine: "postgres", container: oldImage + " true db1"}

	_, err := deploy(docker, srv, deployRequest())

	var failure command.Failure
	if !errors.As(err, &failure) || failure.Code != enums.ErrorCodeBusy {
		t.Fatalf("err = %v", err)
	}
	for _, verb := range []string{"stop", "rm", "run", "start"} {
		if containsCall(calls, verb+" ") {
			t.Errorf("%s ran: %v", verb, calls)
		}
	}
	if marker, _ := os.ReadFile(filepath.Join(srv, "postgres", ".database")); string(marker) != "other" {
		t.Errorf(".database was overwritten with %q", marker)
	}
}

func TestDeployRefusesAnImageWithoutTheEngineLabel(t *testing.T) {
	docker := fakeDocker{calls: &[]string{}, pulled: true, engine: ""}

	_, err := deploy(docker, t.TempDir(), deployRequest())

	var failure command.Failure
	if !errors.As(err, &failure) || failure.Code != enums.ErrorCodeInvalidRequest {
		t.Fatalf("err = %v", err)
	}
}

func TestDeployRefusesAnImageNotBuiltOnTheSdk(t *testing.T) {
	calls := []string{}
	docker := fakeDocker{calls: &calls, pulled: true, engine: "postgres", container: oldImage + " true db1", withoutSdk: true}

	_, err := deploy(docker, t.TempDir(), deployRequest())

	var failure command.Failure
	if !errors.As(err, &failure) || failure.Code != enums.ErrorCodeInvalidRequest || failure.Message != newImage+" is not built on the Databasify engine SDK" {
		t.Fatalf("err = %v", err)
	}
	for _, verb := range []string{"stop", "rm", "run", "start"} {
		if containsCall(calls, verb+" ") {
			t.Errorf("%s ran: %v", verb, calls)
		}
	}
}

func busy(err error) bool {
	var failure command.Failure

	return errors.As(err, &failure) && failure.Code == enums.ErrorCodeBusy
}

func TestDeployRefusesAForeignMarkerEvenWithEmptyData(t *testing.T) {
	srv, calls := t.TempDir(), []string{}
	if err := os.MkdirAll(filepath.Join(srv, "postgres", "data"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srv, "postgres", ".database"), []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := deploy(fakeDocker{calls: &calls, pulled: true, engine: "postgres"}, srv, deployRequest())

	if !busy(err) {
		t.Fatalf("err = %v", err)
	}
	if marker, _ := os.ReadFile(filepath.Join(srv, "postgres", ".database")); string(marker) != "other" {
		t.Errorf(".database was overwritten with %q", marker)
	}
}

func TestDeployRefusesAContainerOfAnotherDatabase(t *testing.T) {
	calls := []string{}
	docker := fakeDocker{calls: &calls, pulled: true, engine: "postgres", container: newImage + " true other"}

	_, err := deploy(docker, t.TempDir(), deployRequest())

	if !busy(err) {
		t.Fatalf("err = %v", err)
	}
	for _, verb := range []string{"stop", "rm", "run", "start"} {
		if containsCall(calls, verb+" ") {
			t.Errorf("%s ran: %v", verb, calls)
		}
	}
}

func TestDeployLeavesTheHostBinaryAlone(t *testing.T) {
	calls := []string{}
	docker := fakeDocker{calls: &calls, engine: "postgres"}

	if _, err := deployAndSettle(docker, t.TempDir(), system{root: t.TempDir(), run: refuseCommands(t)}, deployRequest()); err != nil {
		t.Fatal(err)
	}

	if containsCall(calls, "cp databasify-postgres:/usr/local/bin/databasify") {
		t.Errorf("the host binary was copied out of the container: %v", calls)
	}
}
