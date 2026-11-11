package nodefs

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"os"
	"strings"

	"github.com/databasifylabs/engine-sdk/command"
	"github.com/databasifylabs/engine-sdk/requests"
)

type tlsFile struct {
	path    string
	content string
	mode    os.FileMode
}

func (r Root) WriteTLS(request requests.TlsInstall, combined bool) (bool, error) {
	if err := checkTLS(request); err != nil {
		return false, err
	}
	if err := r.Dir("tls"); err != nil {
		return false, err
	}
	changed := false
	for _, file := range tlsFilesOf(request, combined) {
		written, err := r.Write(file.path, file.content, file.mode, true)
		if err != nil {
			return false, err
		}
		changed = changed || written
	}

	return changed, nil
}

func (r Root) InstallTLS(ctx context.Context, request requests.TlsInstall, combined bool, reload func(context.Context) error) error {
	changed, err := r.WriteTLS(request, combined)
	if err != nil || !changed {
		return err
	}

	return reload(ctx)
}

func checkTLS(request requests.TlsInstall) error {
	if _, err := tls.X509KeyPair([]byte(request.Certificate), []byte(request.Key)); err != nil {
		return command.Invalid("the certificate and key are not a matching PEM pair: " + err.Error())
	}
	if block, _ := pem.Decode([]byte(request.Ca)); block == nil {
		return command.Invalid("the CA is not PEM")
	}

	return nil
}

func tlsFilesOf(request requests.TlsInstall, combined bool) []tlsFile {
	ca := tlsFile{"tls/ca.crt", request.Ca, 0o644}
	if combined {
		return []tlsFile{{"tls/server.pem", withNewline(request.Certificate) + request.Key, 0o600}, ca}
	}

	return []tlsFile{{"tls/server.crt", request.Certificate, 0o644}, {"tls/server.key", request.Key, 0o600}, ca}
}

func withNewline(text string) string {
	if strings.HasSuffix(text, "\n") {
		return text
	}

	return text + "\n"
}
