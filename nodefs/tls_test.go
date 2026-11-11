package nodefs

import (
	"crypto/tls"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/enums"
	"github.com/databasifylabs/engine-sdk/requests"
	"github.com/databasifylabs/engine-sdk/testtls"
)

func pairOf(t *testing.T) (string, string) {
	t.Helper()
	certificate, key, err := testtls.Pair()
	if err != nil {
		t.Fatal(err)
	}

	return certificate, key
}

func tlsRoot(t *testing.T) Root {
	t.Helper()
	root := Root(t.TempDir())
	if err := root.Dir("tls"); err != nil {
		t.Fatal(err)
	}

	return root
}

func modeOf(t *testing.T, root Root, relative string) os.FileMode {
	t.Helper()
	info, err := os.Stat(root.Path(relative))
	if err != nil {
		t.Fatal(err)
	}

	return info.Mode().Perm()
}

func invalidRequest(err error) bool {
	var failure command.Failure

	return errors.As(err, &failure) && failure.Code == enums.ErrorCodeInvalidRequest
}

func TestWriteTlsSplitsCertificateAndKey(t *testing.T) {
	root := tlsRoot(t)
	certificate, key := pairOf(t)

	if _, err := root.WriteTLS(requests.TlsInstall{Certificate: certificate, Key: key, Ca: certificate}, false); err != nil {
		t.Fatal(err)
	}

	if root.Read("tls/server.crt") != certificate || root.Read("tls/server.key") != key || root.Read("tls/ca.crt") != certificate || root.Exists("tls/server.pem") {
		t.Fatal("the split files are wrong")
	}
	if mode := modeOf(t, root, "tls/server.key"); mode != 0o600 {
		t.Fatalf("server.key mode = %o", mode)
	}
}

func TestWriteTlsCombinesIntoOnePem(t *testing.T) {
	root := tlsRoot(t)
	certificate, key := pairOf(t)

	if _, err := root.WriteTLS(requests.TlsInstall{Certificate: certificate, Key: key, Ca: certificate}, true); err != nil {
		t.Fatal(err)
	}

	if root.Read("tls/server.pem") != certificate+key || root.Read("tls/ca.crt") != certificate || root.Exists("tls/server.key") {
		t.Fatal("the combined files are wrong")
	}
	if mode := modeOf(t, root, "tls/server.pem"); mode != 0o600 {
		t.Fatalf("server.pem mode = %o", mode)
	}
}

func TestWriteTlsRefusesAMismatchedKey(t *testing.T) {
	root := tlsRoot(t)
	certificate, key := pairOf(t)
	_, otherKey := pairOf(t)
	if _, err := root.WriteTLS(requests.TlsInstall{Certificate: certificate, Key: key, Ca: certificate}, false); err != nil {
		t.Fatal(err)
	}

	_, err := root.WriteTLS(requests.TlsInstall{Certificate: certificate, Key: otherKey, Ca: certificate}, false)

	if !invalidRequest(err) {
		t.Fatalf("err = %v", err)
	}
	if root.Read("tls/server.key") != key {
		t.Fatal("the installed key was replaced")
	}
}

func TestWriteTlsRefusesTextThatIsNotPem(t *testing.T) {
	_, key := pairOf(t)

	_, err := tlsRoot(t).WriteTLS(requests.TlsInstall{Certificate: "hello", Key: key, Ca: "hello"}, false)

	if !invalidRequest(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteTlsReportsNoChangeForTheSameFiles(t *testing.T) {
	root := tlsRoot(t)
	certificate, key := pairOf(t)
	request := requests.TlsInstall{Certificate: certificate, Key: key, Ca: certificate}
	if _, err := root.WriteTLS(request, false); err != nil {
		t.Fatal(err)
	}

	changed, err := root.WriteTLS(request, false)

	if err != nil || changed {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
}

func TestWriteTlsCreatesTheTlsDirectory(t *testing.T) {
	root := Root(t.TempDir())
	certificate, key := pairOf(t)

	if _, err := root.WriteTLS(requests.TlsInstall{Certificate: certificate, Key: key, Ca: certificate}, true); err != nil {
		t.Fatal(err)
	}

	if root.Read("tls/server.pem") != certificate+key {
		t.Fatal("server.pem was not written")
	}
}

func TestWriteTlsCombinesACertificateWithoutATrailingNewline(t *testing.T) {
	root := tlsRoot(t)
	certificate, key := pairOf(t)

	if _, err := root.WriteTLS(requests.TlsInstall{Certificate: strings.TrimRight(certificate, "\n"), Key: key, Ca: certificate}, true); err != nil {
		t.Fatal(err)
	}

	combined := []byte(root.Read("tls/server.pem"))
	if _, err := tls.X509KeyPair(combined, combined); err != nil {
		t.Fatalf("server.pem is not a valid pair: %v", err)
	}
}
