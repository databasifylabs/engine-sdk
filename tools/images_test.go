package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const commit = "abcdef1234567890abcdef1234567890abcdef12"

func fixtureEngine(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "engine")
	if err := os.CopyFS(dir, os.DirFS("testdata/engine")); err != nil {
		t.Fatal(err)
	}

	return dir
}

func fakeTools(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "docker"), `#!/usr/bin/env bash
[ "$1 $2" = "buildx build" ] || exit 1
printf '%s\n' "$@"
`)
	write(t, filepath.Join(dir, "curl"), "#!/usr/bin/env bash\necho 404\n")
	for _, name := range []string{"docker", "curl"} {
		if err := os.Chmod(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func write(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, engine string, stdin string, args ...string) (string, error) {
	t.Helper()
	script, err := filepath.Abs("images.sh")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = engine
	cmd.Env = append(os.Environ(), "PATH="+fakeTools(t)+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()

	return strings.TrimSpace(string(out)), err
}

func mustRun(t *testing.T, engine string, args ...string) string {
	t.Helper()
	out, err := run(t, engine, "", args...)
	if err != nil {
		t.Fatalf("%v: %v: %s", args, err, out)
	}

	return out
}

func versionsWith(t *testing.T, key string, line string, release string, tag string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "versions.json")
	engine := strings.SplitN(key, "-", 2)[0]
	write(t, path, `{"sdk": "1.0.0", "images": {"`+key+`": {"engine": "`+engine+`", "line": "`+line+`", "release": "`+release+`", "base": {"image": "`+engine+`", "tag": "`+tag+`", "digest": "sha256:0"}}}}`)

	return path
}

func planField(t *testing.T, plan string, name string) string {
	t.Helper()
	for line := range strings.SplitSeq(plan, "\n") {
		if value, found := strings.CutPrefix(line, name+"="); found {
			return value
		}
	}
	t.Fatalf("no %s in the plan: %s", name, plan)

	return ""
}

func TestCheckAcceptsTheCallersPins(t *testing.T) {
	mustRun(t, fixtureEngine(t), "check")
}

func TestCheckRefusesAPinOutsideItsLine(t *testing.T) {
	engine := fixtureEngine(t)
	cases := map[string][4]string{
		"redis 7.2 moved to 7.4":    {"redis-7.2", "7.2", "7.4.0", "7.4.0"},
		"redis 8 moved to 9":        {"redis-8", "8", "9.0.0", "9.0.0"},
		"mariadb 10.11 moved to 11": {"mariadb-10.11", "10.11", "11.4.13", "11.4.13"},
		"postgres 16 moved to 17":   {"postgres-16", "16", "17.0", "17.0-trixie"},
		"key names another line":    {"valkey-8", "9", "9.1.3", "9.1.3"},
		"tag differs from release":  {"mysql-8.4", "8.4", "8.4.13", "8.4.14"},
	}
	for name, pin := range cases {
		if out, err := run(t, engine, "", "--versions", versionsWith(t, pin[0], pin[1], pin[2], pin[3]), "check"); err == nil {
			t.Errorf("%s was accepted: %s", name, out)
		}
	}
}

func TestCheckRefusesAnSdkPinOutsideOne(t *testing.T) {
	engine := fixtureEngine(t)
	for _, pin := range []string{"0.2.2", "2.0.0", "v1.0.0"} {
		versions := filepath.Join(engine, "versions.json")
		content, err := os.ReadFile(versions)
		if err != nil {
			t.Fatal(err)
		}
		write(t, versions, strings.Replace(string(content), `"sdk": "1.0.0"`, `"sdk": "`+pin+`"`, 1))

		if out, err := run(t, engine, "", "check"); err == nil || !strings.Contains(out, pin) {
			t.Errorf("the sdk pin %s was accepted (%v): %s", pin, err, out)
		}
		write(t, versions, string(content))
	}
}

func TestRefreshTakesTheNewestReleaseOfTheLineOnly(t *testing.T) {
	engine := fixtureEngine(t)
	cases := map[string]struct {
		pin  [4]string
		tags string
		want string
	}{
		"redis 7.2 takes patches, never 7.4": {[4]string{"redis-7.2", "7.2", "7.2.17", "7.2.17"}, "7.2.9\n7.2.17\n7.2.18\n7.4.0\n7.4.6\n8.0.0\n7.2.19-rc1\n", "7.2.18"},
		"redis 8 takes newer minors":         {[4]string{"redis-8", "8", "8.10.3", "8.10.3"}, "8.2.9\n8.10.3\n8.12.0\n9.0.0\n8.14.0-m01\n", "8.12.0"},
		"valkey 7 stays on 7":                {[4]string{"valkey-7", "7", "7.2.15", "7.2.15"}, "7.2.15\n7.2.16\n8.0.0\n8.1.11\n", "7.2.16"},
		"mariadb 10.11 takes patches":        {[4]string{"mariadb-10.11", "10.11", "10.11.19", "10.11.19"}, "10.11.19\n10.11.20\n10.2.44\n11.4.13\n10.11.21-ubi\n", "10.11.20"},
		"mongodb 7.0 takes patches":          {[4]string{"mongodb-7.0", "7.0", "7.0.45", "7.0.45"}, "7.0.45\n7.0.46\n7.1.0\n8.0.34\n", "7.0.46"},
		"postgres 16 keeps its distro":       {[4]string{"postgres-16", "16", "16.15", "16.15-trixie"}, "16.15-trixie\n16.16-trixie\n16.16-bookworm\n17.11-trixie\n16.17\n", "16.16-trixie"},
	}
	for name, c := range cases {
		out, err := run(t, engine, c.tags, "--versions", versionsWith(t, c.pin[0], c.pin[1], c.pin[2], c.pin[3]), "newest", c.pin[0])
		if err != nil || out != c.want {
			t.Errorf("%s: got %q (%v), want %s", name, out, err, c.want)
		}
	}
}

func TestKeysListOnlyTheLinesOfTheEngine(t *testing.T) {
	engine := fixtureEngine(t)

	if out := mustRun(t, engine, "keys", "example"); out != `["example-1","example-2"]` {
		t.Errorf("keys of example: %s", out)
	}
	if out := mustRun(t, engine, "keys", "other"); out != `[]` {
		t.Errorf("keys of another engine: %s", out)
	}
}

func TestArgsLabelEveryImageWithTheSdkVersion(t *testing.T) {
	out := mustRun(t, fixtureEngine(t), "args", "example-2")

	if !strings.Contains("\n"+out+"\n", "\nSDK_VERSION=1.0.0\n") {
		t.Fatalf("no SDK_VERSION=1.0.0 in %s", out)
	}
}

func TestPlanPublishesReleaseTagsOnlyFromTheDefaultBranch(t *testing.T) {
	engine := fixtureEngine(t)
	cases := map[string]struct {
		key           string
		defaultBranch string
		want          string
	}{
		"newest line on the default branch": {"example-2", "true", "2.1.0-abcdef1,2.1.0,2,latest"},
		"older line on the default branch":  {"example-1", "true", "1.4.2-abcdef1,1.4.2,1"},
		"newest line on another branch":     {"example-2", "false", "2.1.0-abcdef1"},
		"older line on another branch":      {"example-1", "false", "1.4.2-abcdef1"},
	}
	for name, c := range cases {
		plan := mustRun(t, engine, "plan", c.key, commit, c.defaultBranch)
		if tags := planField(t, plan, "tags"); tags != c.want {
			t.Errorf("%s: tags %s, want %s", name, tags, c.want)
		}
		if repository := planField(t, plan, "repository"); repository != "ghcr.io/example/databasify-example" {
			t.Errorf("%s: repository %s", name, repository)
		}
	}
}

func TestBuildUsesTheCallersDockerfileWithoutAModulesContext(t *testing.T) {
	out := mustRun(t, fixtureEngine(t), "build", "example-2")

	if strings.Contains(out, "--build-context") {
		t.Errorf("the build passed a build context: %s", out)
	}
	if !strings.Contains(out, filepath.Join("images", "example", "Dockerfile")) {
		t.Errorf("the build does not use the caller's Dockerfile: %s", out)
	}
}

func TestHashFollowsTheSdkPin(t *testing.T) {
	engine := fixtureEngine(t)
	before := mustRun(t, engine, "hash", "example-2")
	versions := filepath.Join(engine, "versions.json")
	content, err := os.ReadFile(versions)
	if err != nil {
		t.Fatal(err)
	}

	write(t, versions, strings.Replace(string(content), `"sdk": "1.0.0"`, `"sdk": "1.0.1"`, 1))

	if after := mustRun(t, engine, "hash", "example-2"); after == before {
		t.Fatalf("the hash ignores the sdk pin: %s", after)
	}
}

func TestHashFollowsTheDockerfile(t *testing.T) {
	engine := fixtureEngine(t)
	before := mustRun(t, engine, "hash", "example-2")

	write(t, filepath.Join(engine, "images", "example", "Dockerfile"), "ARG BASE_IMAGE\nFROM ${BASE_IMAGE}\nRUN true\n")

	if after := mustRun(t, engine, "hash", "example-2"); after == before {
		t.Fatalf("the hash ignores the Dockerfile: %s", after)
	}
}
