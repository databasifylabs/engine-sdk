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
	request, err := DecodeStrict[requests.UserCreate](strings.NewReader(`{"name":"app","password":"p'a\"s","grants":[{"database":"shop","access":"write"}],"encryption":null}`))
	if err != nil || request.Name != "app" || request.Password != `p'a"s` || request.Grants[0].Database != "shop" || request.Grants[0].Access != enums.AccessWrite || request.Encryption != nil {
		t.Fatalf("request = %+v, err = %v", request, err)
	}
}

func TestDecodeStrictRejectsInvalidRequests(t *testing.T) {
	cases := map[string]error{
		"unknown field":      decodeError[requests.DatabaseCreate](`{"name":"shop","extra":1}`),
		"case-variant key":   decodeError[requests.DatabaseCreate](`{"Name":"shop"}`),
		"missing field":      decodeError[requests.DatabaseCreate](`{}`),
		"null in required":   decodeError[requests.DatabaseCreate](`{"name":null}`),
		"pattern":            decodeError[requests.DatabaseCreate](`{"name":"a b'; --"}`),
		"too long":           decodeError[requests.DatabaseCreate](`{"name":"` + strings.Repeat("a", 65) + `"}`),
		"empty password":     decodeError[requests.UserSetPassword](`{"name":"app","password":""}`),
		"list item pattern":  decodeError[requests.UserGrant](`{"name":"app","grants":[{"database":"bad name","access":"write"}],"replication":false}`),
		"invalid enum":       decodeError[responses.NodeStatus](`{"role":"leader","version":null}`),
		"negative minimum":   decodeError[dtos.Progress](`{"done":-5,"total":null,"unit":"bytes"}`),
		"not an object":      decodeError[requests.DatabaseCreate](`[]`),
		"nested unknown key": decodeError[requests.NodeConfigure](`{"fresh":true,"clustered":false,"readOnly":false,"ip":"10.0.0.1","databaseId":"d","members":[{"ip":"10.0.0.1","position":1,"voting":true,"x":1}],"settings":{},"secrets":{},"pools":[],"maxClientConnections":1,"defaultPoolSize":1,"archive":null}`),
	}
	for name, err := range cases {
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestDecodeStrictAllowsAbsentOrNullOptionalFields(t *testing.T) {
	for _, input := range []string{`{"name":"app","password":"x"}`, `{"name":"app","password":"x","encryption":null}`} {
		if err := decodeError[requests.UserSetPassword](input); err != nil {
			t.Fatalf("%s: %v", input, err)
		}
	}
}

func TestEncodeWritesEmptyListsAndMaps(t *testing.T) {
	encoded, err := Encode(responses.AccountList{})
	if err != nil || string(encoded) != `{"users":[],"databases":[],"grants":{}}` {
		t.Fatalf("encoded = %s, err = %v", encoded, err)
	}
	encoded, _ = Encode(responses.Error{Error: dtos.Error{Code: enums.ErrorCodeInternal, Message: "x"}})
	if !strings.Contains(string(encoded), `"details":{}`) {
		t.Fatalf("encoded = %s", encoded)
	}
}

func TestMigrationStartAllowsNullOwner(t *testing.T) {
	input := `{"source":{"host":"db","port":5432,"username":"u","password":"p","tls":true},"databases":[],"continuous":false,"owner":null,"publication":null}`
	request, err := DecodeStrict[requests.MigrationStart](strings.NewReader(input))
	if err != nil || request.Owner != nil || request.Publication != nil {
		t.Fatalf("request = %+v, err = %v", request, err)
	}
}
