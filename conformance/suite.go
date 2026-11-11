package conformance

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/databasifylabs/engine-sdk/contract"
	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/responses"
	"github.com/databasifylabs/engine-sdk/support"
	"github.com/databasifylabs/engine-sdk/testtls"
)

const (
	commandTimeout  = 60 * time.Second
	quotedPassword  = "p'a\"s\\s wörd"
	changedPassword = "n'e\"w\\p wörd"
	newlinePassword = "l'i\"n\\e\nwörd"
	codeNotFound    = "not_found"
)

var (
	mandatoryCommands = []string{"engine capabilities", "engine host-profile", "engine info", "node status"}
	passwordForms     = formsOf(quotedPassword, changedPassword, newlinePassword)
)

type Failure struct {
	Step string
	Err  error
}

type step struct {
	name    string
	command string
	run     func() error
}

type lifecycle struct {
	ctx             context.Context
	runner          Runner
	database        string
	user            string
	missingUser     string
	missingKey      string
	capabilities    responses.EngineCapabilities
	databaseCreated bool
	userCreated     bool
	failures        []Failure
}

func Run(t *testing.T, r Runner) {
	t.Helper()
	l := newLifecycle(context.Background(), r)
	t.Cleanup(l.cleanup)
	for _, failure := range l.run() {
		t.Errorf("%s: %v", failure.Step, failure.Err)
	}
}

func Failures(ctx context.Context, r Runner) []Failure {
	return newLifecycle(ctx, r).run()
}

func newLifecycle(ctx context.Context, r Runner) *lifecycle {
	suffix := randomHex(3)
	return &lifecycle{
		ctx:         ctx,
		runner:      r,
		database:    "ct_" + suffix,
		user:        "ct_" + suffix,
		missingUser: "ct_missing_" + suffix,
		missingKey:  "ct-missing-" + suffix,
	}
}

func randomHex(size int) string {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buffer)
}

func (l *lifecycle) run() []Failure {
	if !l.discoverCapabilities() {
		return l.failures
	}
	for _, s := range l.steps() {
		l.perform(s)
	}
	l.probeUnsupported()
	l.cleanup()
	return l.failures
}

func (l *lifecycle) fail(name string, err error) {
	l.failures = append(l.failures, Failure{Step: name, Err: err})
}

func (l *lifecycle) supports(command string) bool {
	return slices.Contains(l.capabilities.Commands, command)
}

func (l *lifecycle) perform(s step) {
	if !l.supports(s.command) {
		return
	}
	if err := s.run(); err != nil {
		l.fail(s.name, err)
	}
}

func (l *lifecycle) discoverCapabilities() bool {
	capabilities, err := callReply[responses.EngineCapabilities](l, "engine capabilities", dtos.Empty{})
	if err != nil {
		l.fail("engine capabilities", err)
		return false
	}
	l.capabilities = capabilities
	for _, command := range mandatoryCommands {
		if !l.supports(command) {
			l.fail(command+" mandatory", fmt.Errorf("commands %v do not list the mandatory %s", capabilities.Commands, command))
		}
	}
	return true
}

func (l *lifecycle) steps() []step {
	return []step{
		{"engine info", "engine info", l.checkEngineInfo},
		{"engine host-profile", "engine host-profile", func() error { return l.expectSuccess("engine host-profile", dtos.Empty{}) }},
		{"node status", "node status", func() error { return l.expectSuccess("node status", dtos.Empty{}) }},
		{"account list", "account list", func() error { return l.expectSuccess("account list", dtos.Empty{}) }},
		{"database create", "database create", func() error { return l.createDatabase(true) }},
		{"database create again", "database create", func() error { return l.createDatabase(false) }},
		{"user create", "user create", func() error { return l.createUser(quotedPassword, true) }},
		{"user create again", "user create", func() error { return l.createUser(changedPassword, false) }},
		{"user set-password", "user set-password", l.setPassword},
		{"user set-password with a newline", "user set-password", l.setNewlinePassword},
		{"user set-password missing user", "user set-password", func() error {
			request := requests.UserSetPassword{Name: l.missingUser, Password: changedPassword}
			return l.expectError("user set-password", request, exitFailure, codeNotFound)
		}},
		{"user grant", "user grant", l.grant},
		{"account list shows new accounts", "account list", l.listAccounts},
		{"user drop", "user drop", func() error { return l.dropUser(true) }},
		{"database drop", "database drop", func() error { return l.dropDatabase(true) }},
		{"user drop again", "user drop", func() error { return l.dropUser(false) }},
		{"database drop again", "database drop", func() error { return l.dropDatabase(false) }},
		{"operation status unknown key", "operation status", func() error {
			return l.expectError("operation status", requests.OperationStatus{Key: l.missingKey}, exitFailure, codeNotFound)
		}},
		{"operation cancel unknown key", "operation cancel", func() error {
			return l.expectError("operation cancel", requests.OperationCancel{Key: l.missingKey}, exitFailure, codeNotFound)
		}},
		{"unknown field on database create", "database create", func() error {
			return l.expectInvalid("database create", fmt.Sprintf(`{"name":%q,"unexpectedField":true}`, l.database))
		}},
		{"database create bad identifier", "database create", func() error {
			return l.expectInvalid("database create", `{"name":"a b'; --"}`)
		}},
		{"database create missing key", "database create", func() error { return l.expectInvalid("database create", `{}`) }},
		{"database create case-variant key", "database create", func() error {
			return l.expectInvalid("database create", fmt.Sprintf(`{"name":%[1]q,"Name":%[1]q}`, l.database))
		}},
		{"user create empty password", "user create", func() error {
			return l.expectInvalid("user create", fmt.Sprintf(`{"name":%q,"password":"","databases":[],"encryption":null}`, l.user))
		}},
		{"engine info array input", "engine info", func() error { return l.expectInvalid("engine info", `[]`) }},
		{"tls install", "tls install", l.installTLS},
		{"tls install mismatched key", "tls install", l.installMismatchedTLS},
	}
}

func (l *lifecycle) probeUnsupported() {
	for _, spec := range contract.Commands {
		if l.supports(spec.Name) {
			continue
		}
		err := l.expectError(spec.Name, l.requestFor(spec.Name), exitFailure, codeUnsupported)
		if err != nil {
			l.fail(spec.Name+" unsupported", err)
		}
	}
}

func (l *lifecycle) requestFor(command string) any {
	switch command {
	case "database create":
		return requests.DatabaseCreate{Name: l.database}
	case "database drop":
		return requests.DatabaseDrop{Name: l.database}
	case "user create":
		return requests.UserCreate{Name: l.user, Password: quotedPassword, Grants: []dtos.Grant{{Database: l.database, Access: enums.AccessWrite}}}
	case "user set-password":
		return requests.UserSetPassword{Name: l.user, Password: changedPassword}
	case "user drop":
		return requests.UserDrop{Name: l.user}
	case "user grant":
		return requests.UserGrant{Name: l.user, Grants: []dtos.Grant{{Database: l.database, Access: enums.AccessWrite}}}
	case "operation status":
		return requests.OperationStatus{Key: l.missingKey}
	case "operation cancel":
		return requests.OperationCancel{Key: l.missingKey}
	}
	return struct{}{}
}

func (l *lifecycle) exec(command string, request any) ([]byte, int, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(l.ctx, commandTimeout)
	defer cancel()
	stdout, exit, err := l.runner.Run(ctx, strings.Fields(command), body)
	if err != nil {
		return nil, 0, fmt.Errorf("runner failed: %w", err)
	}
	if err := Check(command, stdout, exit); err != nil {
		return nil, 0, err
	}
	return stdout, exit, rejectEchoedPasswords(stdout)
}

func formsOf(passwords ...string) [][]byte {
	var forms [][]byte
	for _, password := range passwords {
		escaped, err := json.Marshal(password)
		if err != nil {
			panic(err)
		}
		forms = append(forms, []byte(password), escaped[1:len(escaped)-1])
	}
	return forms
}

func rejectEchoedPasswords(stdout []byte) error {
	for _, form := range passwordForms {
		if bytes.Contains(stdout, form) {
			return errors.New("stdout contains a test password")
		}
	}
	return nil
}

func (l *lifecycle) call(command string, request any) ([]byte, error) {
	stdout, exit, err := l.exec(command, request)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, fmt.Errorf("exit = %d, want 0; stdout: %s", exit, rawOutput(stdout))
	}
	return stdout, nil
}

func callReply[T any](l *lifecycle, command string, request any) (T, error) {
	var zero T
	stdout, err := l.call(command, request)
	if err != nil {
		return zero, err
	}
	return support.DecodeStrict[T](bytes.NewReader(stdout))
}

func (l *lifecycle) expectSuccess(command string, request any) error {
	_, err := l.call(command, request)
	return err
}

func (l *lifecycle) installTLS() error {
	certificate, key, err := testtls.Pair()
	if err != nil {
		return err
	}

	return l.expectSuccess("tls install", requests.TlsInstall{Certificate: certificate, Key: key, Ca: certificate})
}

func (l *lifecycle) installMismatchedTLS() error {
	certificate, _, err := testtls.Pair()
	if err != nil {
		return err
	}
	_, otherKey, err := testtls.Pair()
	if err != nil {
		return err
	}

	return l.expectError("tls install", requests.TlsInstall{Certificate: certificate, Key: otherKey, Ca: certificate}, exitInvalid, codeInvalid)
}

func (l *lifecycle) expectInvalid(command, request string) error {
	return l.expectError(command, json.RawMessage(request), exitInvalid, codeInvalid)
}

func (l *lifecycle) expectError(command string, request any, wantExit int, wantCode string) error {
	stdout, exit, err := l.exec(command, request)
	if err != nil {
		return err
	}
	if exit != wantExit {
		return fmt.Errorf("exit = %d, want %d; stdout: %s", exit, wantExit, rawOutput(stdout))
	}
	code, err := errorCode(stdout)
	if err != nil {
		return err
	}
	if code != wantCode {
		return fmt.Errorf("error code = %q, want %q", code, wantCode)
	}
	return nil
}

func expectFlag(name string, got bool, want bool) error {
	if got != want {
		return fmt.Errorf("%s = %v, want %v", name, got, want)
	}
	return nil
}

func (l *lifecycle) createDatabase(want bool) error {
	reply, err := callReply[responses.DatabaseCreate](l, "database create", l.requestFor("database create"))
	if err != nil {
		return err
	}
	l.databaseCreated = l.databaseCreated || reply.Created
	return expectFlag("created", reply.Created, want)
}

func (l *lifecycle) checkEngineInfo() error {
	reply, err := callReply[responses.EngineInfo](l, "engine info", dtos.Empty{})
	if err != nil {
		return err
	}
	if reply.Sdk == "" {
		return fmt.Errorf("engine info reports no sdk version")
	}
	return nil
}

func (l *lifecycle) createUser(password string, want bool) error {
	request := requests.UserCreate{Name: l.user, Password: password, Grants: []dtos.Grant{{Database: l.database, Access: enums.AccessWrite}}}
	reply, err := callReply[responses.UserCreate](l, "user create", request)
	if err != nil {
		return err
	}
	l.userCreated = l.userCreated || reply.Created
	return expectFlag("created", reply.Created, want)
}

func (l *lifecycle) setPassword() error {
	_, err := callReply[dtos.Empty](l, "user set-password", l.requestFor("user set-password"))
	return err
}

func (l *lifecycle) setNewlinePassword() error {
	stdout, exit, err := l.exec("user set-password", requests.UserSetPassword{Name: l.user, Password: newlinePassword})
	if err != nil || exit == 0 {
		return err
	}
	code, err := errorCode(stdout)
	if err != nil {
		return err
	}
	if exit != exitInvalid || code != codeInvalid {
		return fmt.Errorf("exit = %d, code = %q; want success or a refused password; stdout: %s", exit, code, rawOutput(stdout))
	}

	return nil
}

func (l *lifecycle) grant() error {
	_, err := callReply[dtos.Empty](l, "user grant", l.requestFor("user grant"))
	return err
}

func (l *lifecycle) listAccounts() error {
	reply, err := callReply[responses.AccountList](l, "account list", dtos.Empty{})
	if err != nil {
		return err
	}
	if !slices.Contains(reply.Users, l.user) {
		return fmt.Errorf("users %v do not contain %s", reply.Users, l.user)
	}
	if !slices.Contains(reply.Databases, l.database) {
		return fmt.Errorf("databases %v do not contain %s", reply.Databases, l.database)
	}
	return nil
}

func (l *lifecycle) dropUser(want bool) error {
	reply, err := callReply[responses.UserDrop](l, "user drop", l.requestFor("user drop"))
	if err != nil {
		return err
	}
	l.userCreated = false
	return expectFlag("dropped", reply.Dropped, want)
}

func (l *lifecycle) dropDatabase(want bool) error {
	reply, err := callReply[responses.DatabaseDrop](l, "database drop", l.requestFor("database drop"))
	if err != nil {
		return err
	}
	l.databaseCreated = false
	return expectFlag("dropped", reply.Dropped, want)
}

func (l *lifecycle) cleanup() {
	if l.userCreated && l.supports("user drop") {
		_ = l.dropUser(true)
		l.userCreated = false
	}
	if l.databaseCreated && l.supports("database drop") {
		_ = l.dropDatabase(true)
		l.databaseCreated = false
	}
}
