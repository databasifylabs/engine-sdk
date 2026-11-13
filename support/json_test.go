package support

import (
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
)

func decodeError[T any](input string) error {
	_, err := DecodeStrict[T](strings.NewReader(input))

	return err
}

func TestDecodeStrictAcceptsAValidRequest(t *testing.T) {
	user := "postgres"
	request, err := DecodeStrict[requests.TaskStart](strings.NewReader(`{"key":"ready.1","script":"echo 'a\"b'","env":{"A":"1"},"user":"postgres"}`))
	if err != nil || request.Key != "ready.1" || request.Script != `echo 'a"b'` || request.Env["A"] != "1" || *request.User != user {
		t.Fatalf("request = %+v, err = %v", request, err)
	}
}

type unitList struct {
	Units []string `json:"units" pattern:"^[a-z-]+$"`
}

func TestDecodeStrictRejectsInvalidRequests(t *testing.T) {
	cases := map[string]error{
		"unknown field":     decodeError[requests.TaskKey](`{"key":"k1","extra":1}`),
		"case-variant key":  decodeError[requests.TaskKey](`{"Key":"k1"}`),
		"missing field":     decodeError[requests.TaskKey](`{}`),
		"null in required":  decodeError[requests.TaskKey](`{"key":null}`),
		"pattern":           decodeError[requests.TaskKey](`{"key":"a b'; --"}`),
		"too long":          decodeError[requests.FileGet](`{"path":"` + strings.Repeat("a", 256) + `"}`),
		"empty script":      decodeError[requests.RuntimeRun](`{"script":"","env":{},"user":null,"timeoutSeconds":10}`),
		"list item pattern": decodeError[unitList](`{"units":["bad unit!"]}`),
		"maximum":           decodeError[requests.RuntimeRun](`{"script":"true","env":{},"user":null,"timeoutSeconds":99999}`),
		"negative minimum":  decodeError[dtos.Progress](`{"done":-5,"total":null,"unit":"bytes"}`),
		"not an object":     decodeError[requests.TaskKey](`[]`),
	}
	for name, err := range cases {
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestDecodeStrictAllowsAbsentOrNullOptionalFields(t *testing.T) {
	for _, input := range []string{`{"path":"conf/x","contentBase64":"","mode":384}`, `{"path":"conf/x","contentBase64":"","mode":384,"owner":null}`} {
		if err := decodeError[requests.FilePut](input); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
	}
}

func TestEncodeWritesEmptyListsAndMaps(t *testing.T) {
	encoded, err := Encode(responses.LogsRead{})
	if err != nil || strings.Contains(string(encoded), "null") {
		t.Fatalf("encoded = %s, err = %v", encoded, err)
	}
	encoded, _ = Encode(responses.Error{Error: dtos.Error{Code: enums.ErrorCodeInternal, Message: "x"}})
	if !strings.Contains(string(encoded), `"details":{}`) {
		t.Fatalf("encoded = %s", encoded)
	}
}
