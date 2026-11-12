package command

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
)

func isInvalid(err error) bool {
	var failure Failure

	return errors.As(err, &failure) && failure.Code == enums.ErrorCodeInvalidRequest
}

func TestPutWritesAtomicallyWithItsMode(t *testing.T) {
	home := t.TempDir()

	err := putFile(home, requests.FilePut{Path: "conf/processes.json", ContentBase64: base64.StdEncoding.EncodeToString([]byte("{}")), Mode: 0o600})

	info, statErr := os.Stat(filepath.Join(home, "conf/processes.json"))
	content, _ := os.ReadFile(filepath.Join(home, "conf/processes.json"))
	if err != nil || statErr != nil || info.Mode().Perm() != 0o600 || string(content) != "{}" {
		t.Fatalf("%v %v %v %q", err, statErr, info, content)
	}
	if _, err := os.Stat(filepath.Join(home, "conf/processes.json.tmp")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("the temporary file was left behind")
	}
}

func TestPathsOutsideTheHomeAreRefused(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	_ = os.Symlink(outside, filepath.Join(home, "escape"))

	for _, path := range []string{"/etc/passwd", "../x", "conf/../../x", "escape/passwd", ""} {
		if err := putFile(home, requests.FilePut{Path: path, ContentBase64: "", Mode: 0o600}); !isInvalid(err) {
			t.Errorf("%q: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outside, "passwd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a file was written through the symlink")
	}
}

func TestGetReadsAFileAndReportsAMissingOne(t *testing.T) {
	home := t.TempDir()
	_ = os.WriteFile(filepath.Join(home, "settings.json"), []byte("on"), 0o600)

	found, err := getFile(home, requests.FileGet{Path: "settings.json"})
	missing, missingErr := getFile(home, requests.FileGet{Path: "conf/nope"})

	if err != nil || !found.Exists || found.ContentBase64 != base64.StdEncoding.EncodeToString([]byte("on")) || missingErr != nil || missing.Exists {
		t.Fatalf("%+v %v %+v %v", found, err, missing, missingErr)
	}
}

func TestPutNeverFollowsAPlantedStagingSymlink(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	_ = os.Symlink(filepath.Join(outside, "victim"), filepath.Join(home, "conf.tmp"))

	_ = putFile(home, requests.FilePut{Path: "conf", ContentBase64: base64.StdEncoding.EncodeToString([]byte("hi")), Mode: 0o600})

	if _, err := os.Stat(filepath.Join(outside, "victim")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("file put wrote through a symlink outside the home")
	}
}
