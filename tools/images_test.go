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
for arg; do
    case $arg in modules=*) (cd "${arg#modules=}" && find . -type f | LC_ALL=C sort | sed 's|^\./|staged |') ;; esac
done
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

func appendTo(t *testing.T, path string, content string) {
	t.Helper()
	existing, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, string(existing)+content)
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
	write(t, path, `{"sdk": "0.2.0", "images": {"`+key+`": {"engine": "`+engine+`", "line": "`+line+`", "release": "`+release+`", "base": {"image": "`+engine+`", "tag": "`+tag+`", "digest": "sha256:0"}}}}`)

	return path
}

func sdkCheckout(t *testing.T, engine string) string {
	t.Helper()
	dir := siblingModule(t, engine, "engine-sdk", "github.com/databasifylabs/engine-sdk")
	write(t, filepath.Join(dir, "sdk.go"), "package sdk\n\nconst Version = \"0.2.0\"\n")

	return dir
}

func siblingModule(t *testing.T, engine string, name string, module string) string {
	t.Helper()
	dir := filepath.Join(filepath.Dir(engine), name)
	write(t, filepath.Join(dir, "go.mod"), "module "+module+"\n\ngo 1.25.0\n")

	return dir
}

func replaceSdk(t *testing.T, engine string, path string) {
	t.Helper()
	replace(t, engine, "github.com/databasifylabs/engine-sdk", path)
}

func replace(t *testing.T, engine string, module string, path string) {
	t.Helper()
	appendTo(t, filepath.Join(engine, "go.mod"), "\nreplace "+module+" => "+path+"\n")
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

func TestCheckRefusesAnSdkVersionGoModDoesNotRequire(t *testing.T) {
	engine := fixtureEngine(t)
	write(t, filepath.Join(engine, "go.mod"), "module example.com/engine-example\n\ngo 1.25.0\n\nrequire github.com/databasifylabs/engine-sdk v0.3.0\n")

	if out, err := run(t, engine, "", "check"); err == nil || !strings.Contains(out, "0.3.0") {
		t.Fatalf("an sdk pin go.mod does not require was accepted (%v): %s", err, out)
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

	if !strings.Contains("\n"+out+"\n", "\nSDK_VERSION=0.2.0\n") {
		t.Fatalf("no SDK_VERSION=0.2.0 in %s", out)
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

func TestBuildPassesAnEmptyModulesContextWithoutAReplacement(t *testing.T) {
	engine := fixtureEngine(t)

	out := mustRun(t, engine, "build", "example-2")

	if !strings.Contains(out, "--build-context\nmodules=") {
		t.Errorf("a build without a replacement passed no modules context: %s", out)
	}
	if strings.Contains(out, "staged ") {
		t.Errorf("a build without a replacement staged files: %s", out)
	}
	if !strings.Contains(out, filepath.Join("images", "example", "Dockerfile")) {
		t.Errorf("the build does not use the caller's Dockerfile: %s", out)
	}
}

func TestBuildStagesTheGoSourcesOfEveryLocalReplacement(t *testing.T) {
	engine := fixtureEngine(t)
	sdk := sdkCheckout(t, engine)
	write(t, filepath.Join(sdk, "sdk_test.go"), "package sdk\n")
	write(t, filepath.Join(sdk, "README.md"), "docs\n")
	other := siblingModule(t, engine, "engine-other", "example.com/engine-other")
	write(t, filepath.Join(other, "family", "family.go"), "package family\n")
	replaceSdk(t, engine, "../engine-sdk")
	replace(t, engine, "example.com/engine-other", "../engine-other")

	out := mustRun(t, engine, "build", "example-2")

	for _, want := range []string{"staged engine-sdk/go.mod", "staged engine-sdk/sdk.go", "staged engine-other/go.mod", "staged engine-other/family/family.go"} {
		if !strings.Contains(out, want+"\n") && !strings.HasSuffix(out, want) {
			t.Errorf("%s missing: %s", want, out)
		}
	}
	for _, unwanted := range []string{"sdk_test.go", "README.md"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("%s was staged: %s", unwanted, out)
		}
	}
}

func TestBuildRefusesAReplacementOutsideTheSiblingLayout(t *testing.T) {
	engine := fixtureEngine(t)
	replaceSdk(t, engine, sdkCheckout(t, engine))

	if out, err := run(t, engine, "", "build", "example-2"); err == nil || !strings.Contains(out, "../<dir>") {
		t.Fatalf("an absolute replacement was accepted (%v): %s", err, out)
	}
}

func TestHashFollowsTheSdkVersionInGoMod(t *testing.T) {
	engine := fixtureEngine(t)
	before := mustRun(t, engine, "hash", "example-2")

	write(t, filepath.Join(engine, "go.mod"), "module example.com/engine-example\n\ngo 1.25.0\n\nrequire github.com/databasifylabs/engine-sdk v0.2.1\n")

	if after := mustRun(t, engine, "hash", "example-2"); after == before {
		t.Fatalf("the hash ignores the sdk version in go.mod: %s", after)
	}
}

func TestHashFollowsTheReplacedSdkSource(t *testing.T) {
	engine := fixtureEngine(t)
	sdk := sdkCheckout(t, engine)
	replaceSdk(t, engine, "../engine-sdk")
	before := mustRun(t, engine, "hash", "example-2")

	write(t, filepath.Join(sdk, "sdk.go"), "package sdk\n\nconst Version = \"0.2.1\"\n")

	if after := mustRun(t, engine, "hash", "example-2"); after == before {
		t.Fatalf("the hash ignores the replaced sdk source: %s", after)
	}
}

func TestHashFollowsEveryReplacedModuleSource(t *testing.T) {
	engine := fixtureEngine(t)
	other := siblingModule(t, engine, "engine-other", "example.com/engine-other")
	replace(t, engine, "example.com/engine-other", "../engine-other")
	before := mustRun(t, engine, "hash", "example-2")

	write(t, filepath.Join(other, "other.go"), "package other\n")

	if after := mustRun(t, engine, "hash", "example-2"); after == before {
		t.Fatalf("the hash ignores a replaced sibling module: %s", after)
	}
}

func TestHashFailsWhenAReplacedModuleIsMissing(t *testing.T) {
	engine := fixtureEngine(t)
	replaceSdk(t, engine, "../missing-engine-sdk")

	for _, args := range [][]string{{"hash", "example-2"}, {"build", "example-2"}, {"plan", "example-2", commit, "true"}} {
		if out, err := run(t, engine, "", args...); err == nil || !strings.Contains(out, "not a module checkout") {
			t.Errorf("%v did not refuse the missing sdk (%v): %s", args, err, out)
		}
	}
}
