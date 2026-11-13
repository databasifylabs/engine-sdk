package conformance

import (
	"strings"
	"testing"
)

const errorReply = `{"error":{"code":"%s","message":"m","details":{}}}`

func errorBody(code string) []byte {
	return []byte(strings.Replace(errorReply, "%s", code, 1))
}

func TestValidReplyPasses(t *testing.T) {
	if err := Check("file get", []byte(`{"contentBase64":"","exists":true}`), 0); err != nil {
		t.Fatal(err)
	}
}

func TestExtraReplyFieldFails(t *testing.T) {
	if err := Check("file get", []byte(`{"contentBase64":"","exists":true,"extra":1}`), 0); err == nil {
		t.Fatal("extra field accepted")
	}
}

func TestErrorReplyWithExitOnePasses(t *testing.T) {
	if err := Check("file get", errorBody("not_found"), 1); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidRequestNeedsExit64Code(t *testing.T) {
	if err := Check("file get", errorBody("invalid_request"), 64); err != nil {
		t.Fatal(err)
	}
	if err := Check("file get", errorBody("not_found"), 64); err == nil {
		t.Fatal("exit 64 accepted a code other than invalid_request")
	}
}

func TestInternalNeedsExit70Code(t *testing.T) {
	if err := Check("file get", errorBody("internal"), 70); err != nil {
		t.Fatal(err)
	}
	if err := Check("file get", errorBody("busy"), 70); err == nil {
		t.Fatal("exit 70 accepted a code other than internal")
	}
}

func TestNonJSONStdoutShowsRawOutput(t *testing.T) {
	err := Check("file get", []byte("panic: boom\ngoroutine 1"), 0)
	if err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("error = %v, want raw output", err)
	}
}

func TestEmptyStdoutFails(t *testing.T) {
	if err := Check("file get", nil, 0); err == nil {
		t.Fatal("empty stdout accepted")
	}
}

func TestRawOutputIsCappedAtTwoKiB(t *testing.T) {
	err := Check("file get", []byte(strings.Repeat("x", 5000)), 0)
	if err == nil || strings.Count(err.Error(), "x") != 2048 {
		t.Fatalf("error = %v, want exactly 2048 raw bytes", err)
	}
}

func TestUnknownExitFails(t *testing.T) {
	for _, exit := range []int{2, 69} {
		if err := Check("file get", errorBody("internal"), exit); err == nil {
			t.Fatalf("exit %d accepted", exit)
		}
	}
}

func TestUnknownCommandFails(t *testing.T) {
	if err := Check("no such", []byte(`{}`), 0); err == nil {
		t.Fatal("unknown command accepted")
	}
}

func TestRawOutputCutsOnRuneBoundary(t *testing.T) {
	body := strings.Repeat("x", 2047) + "ö" + "tail"
	err := Check("file get", []byte(body), 0)
	if err == nil || !strings.Contains(err.Error(), strings.Repeat("x", 2047)) || strings.Contains(err.Error(), "�") || strings.Contains(err.Error(), "ö") {
		t.Fatalf("error = %v, want 2047 bytes cut before the split rune", err)
	}
}

func TestExitOneWithInvalidRequestFails(t *testing.T) {
	if err := Check("file get", errorBody("invalid_request"), 1); err == nil {
		t.Fatal("exit 1 accepted invalid_request")
	}
}

func TestExitOneWithInternalFails(t *testing.T) {
	if err := Check("file get", errorBody("internal"), 1); err == nil {
		t.Fatal("exit 1 accepted internal")
	}
}
