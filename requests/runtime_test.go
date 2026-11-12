package requests_test

import (
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/support"
)

func TestRuntimeRunDecodesStrictly(t *testing.T) {
	_, err := support.DecodeStrict[requests.RuntimeRun](strings.NewReader(`{"script":"echo hi","env":{},"user":null,"timeoutSeconds":30}`))
	if err != nil {
		t.Fatal(err)
	}
	_, err = support.DecodeStrict[requests.RuntimeRun](strings.NewReader(`{"script":"echo hi","env":{},"user":null,"timeoutSeconds":30,"shell":"zsh"}`))
	if err == nil || !strings.Contains(err.Error(), `unknown field "shell"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestTaskKeysFollowTheFlowKeyShape(t *testing.T) {
	for key, valid := range map[string]bool{"migration.copy.1": true, "a": true, "Upper": false, "../x": false, "": false} {
		_, err := support.DecodeStrict[requests.TaskKey](strings.NewReader(`{"key":"` + key + `"}`))
		if (err == nil) != valid {
			t.Errorf("key %q: err = %v", key, err)
		}
	}
}

func TestRuntimeRunRefusesATimeoutOverAnHour(t *testing.T) {
	_, err := support.DecodeStrict[requests.RuntimeRun](strings.NewReader(`{"script":"echo hi","env":{},"user":null,"timeoutSeconds":3601}`))
	if err == nil {
		t.Fatal("a 3601-second timeout was accepted")
	}
}
