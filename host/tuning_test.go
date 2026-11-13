package host

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/responses"
)

func refuseCommands(t *testing.T) func(string, ...string) error {
	return func(name string, args ...string) error {
		t.Errorf("ran %s %v", name, args)

		return nil
	}
}

func recordCommands(calls *[]string) func(string, ...string) error {
	return func(name string, args ...string) error {
		*calls = append(*calls, name+" "+strings.Join(args, " "))

		return nil
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

func TestFloorsKeepAHigherCurrentValue(t *testing.T) {
	profile := responses.EngineHostProfile{Settings: map[string]string{"vm.overcommit_memory": "1"}, Floors: map[string]int64{"net.core.somaxconn": 1024}}

	settings := settingsOf(profile, func(string) int64 { return 4096 })

	if settings["net.core.somaxconn"] != "4096" || settings["vm.overcommit_memory"] != "1" {
		t.Fatalf("settings = %v", settings)
	}
}

func TestFloorsRaiseALowCurrentValue(t *testing.T) {
	profile := responses.EngineHostProfile{Floors: map[string]int64{"net.core.somaxconn": 1024}}

	settings := settingsOf(profile, func(string) int64 { return 128 })

	if settings["net.core.somaxconn"] != "1024" {
		t.Fatalf("settings = %v", settings)
	}
}

func TestAZeroProfileHasNoSettings(t *testing.T) {
	if settings := settingsOf(responses.EngineHostProfile{}, func(string) int64 { return 0 }); len(settings) != 0 {
		t.Fatalf("settings = %v", settings)
	}
}

func TestHostTuneWithAZeroProfileChangesNothing(t *testing.T) {
	root := t.TempDir()

	tuned, err := (system{root: root, run: refuseCommands(t)}).tune("fake", responses.EngineHostProfile{})

	if err != nil || len(tuned.Settings) != 0 {
		t.Fatalf("tuned = %+v, err = %v", tuned, err)
	}
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Fatalf("the root was changed: %v", entries)
	}
}

func TestHostTuneAppliesTheEngineProfile(t *testing.T) {
	root, calls := t.TempDir(), []string{}
	writeFile(t, filepath.Join(root, "proc/sys/net/core/somaxconn"), "128\n")
	writeFile(t, filepath.Join(root, "proc/sys/vm/overcommit_memory"), "0\n")
	writeFile(t, filepath.Join(root, "sys/kernel/mm/transparent_hugepage/enabled"), "always [madvise] never\n")
	writeFile(t, filepath.Join(root, "sys/kernel/mm/transparent_hugepage/defrag"), "[always] defer madvise never\n")
	if err := os.MkdirAll(filepath.Join(root, "etc/sysctl.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "etc/systemd/system"), 0o755); err != nil {
		t.Fatal(err)
	}
	profile := responses.EngineHostProfile{Settings: map[string]string{"vm.overcommit_memory": "1"}, Floors: map[string]int64{"net.core.somaxconn": 1024}, DisableTransparentHugepages: true}

	tuned, err := (system{root: root, run: recordCommands(&calls)}).tune("fake", profile)

	if err != nil || tuned.Settings["net.core.somaxconn"] != "1024" || tuned.Settings["vm.overcommit_memory"] != "1" {
		t.Fatalf("tuned = %+v, err = %v", tuned, err)
	}
	if got := readFile(t, filepath.Join(root, "etc/sysctl.d/99-fake.conf")); got != "net.core.somaxconn = 1024\nvm.overcommit_memory = 1\n" {
		t.Errorf("sysctl file = %q", got)
	}
	if got := readFile(t, filepath.Join(root, "etc/databasify/sysctl-originals/fake")); got != "net.core.somaxconn = 128\nvm.overcommit_memory = 0\n" {
		t.Errorf("originals = %q", got)
	}
	if got := readFile(t, filepath.Join(root, "etc/databasify/thp-originals")); got != "enabled=madvise\ndefrag=always\n" {
		t.Errorf("thp originals = %q", got)
	}
	if readFile(t, filepath.Join(root, "etc/systemd/system/disable-thp.service")) != thpUnitText {
		t.Error("the THP unit was not written")
	}
	want := []string{"sysctl --system", "systemctl daemon-reload", "systemctl enable --now disable-thp.service"}
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestDockerErrorsCarryStderr(t *testing.T) {
	bin := t.TempDir()
	writeFile(t, filepath.Join(bin, "docker"), "#!/bin/sh\necho 'usage: databasify <command>' >&2\nexit 64\n")
	if err := os.Chmod(filepath.Join(bin, "docker"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	_, err := execDocker{}.RunWithInput("{}", "exec", "--interactive", "databasify-fake", "databasify", "engine", "host-profile")

	if err == nil || !strings.Contains(err.Error(), "exit status 64") || !strings.Contains(err.Error(), "usage: databasify <command>") {
		t.Fatalf("err = %v", err)
	}
}

func TestHostCommandsAreRecognised(t *testing.T) {
	if !IsHostCommand([]string{"host", "deploy"}) || !IsHostCommand([]string{"host", "remove"}) || IsHostCommand([]string{"host", "tune"}) || IsHostCommand([]string{"host", "restore"}) || IsHostCommand([]string{"node", "status"}) {
		t.Fatal("host command detection is wrong")
	}
}

func TestTuningRecordsTheOriginalOfAKeyAddedLater(t *testing.T) {
	root, calls := t.TempDir(), []string{}
	writeFile(t, filepath.Join(root, "proc/sys/vm/swappiness"), "60\n")
	writeFile(t, filepath.Join(root, "proc/sys/net/core/somaxconn"), "128\n")
	if err := os.MkdirAll(filepath.Join(root, "etc/sysctl.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	host := system{root: root, run: recordCommands(&calls)}
	if _, err := host.tune("fake", responses.EngineHostProfile{Settings: map[string]string{"vm.swappiness": "1"}}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "proc/sys/vm/swappiness"), "1\n")

	if _, err := host.tune("fake", responses.EngineHostProfile{Settings: map[string]string{"vm.swappiness": "1", "net.core.somaxconn": "4096"}}); err != nil {
		t.Fatal(err)
	}

	if got := readFile(t, filepath.Join(root, "etc/databasify/sysctl-originals/fake")); got != "net.core.somaxconn = 128\nvm.swappiness = 60\n" {
		t.Errorf("originals = %q", got)
	}
}
