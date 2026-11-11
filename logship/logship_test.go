package logship

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
)

func appendLine(t *testing.T, path string, line string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(line); err != nil {
		t.Fatal(err)
	}
}

func startShipping(t *testing.T, destination dtos.LogDestination) string {
	t.Helper()
	dir := t.TempDir()
	appendLine(t, filepath.Join(dir, "valkey-server.log"), "old line\n")
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go Run(ctx, destination, dir)
	time.Sleep(1500 * time.Millisecond)
	appendLine(t, filepath.Join(dir, "valkey-server.log"), "hello\n")

	return dir
}

func TestShipsNewLinesToSyslog(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	received := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		line, _ := bufio.NewReader(conn).ReadString('\n')
		received <- line
	}()

	startShipping(t, dtos.LogDestination{Kind: enums.LogDestinationRsyslog, Endpoint: "tcp://" + listener.Addr().String()})

	select {
	case line := <-received:
		if !strings.HasPrefix(line, "<14>1 ") || !strings.HasSuffix(line, " - valkey-server - - - hello\n") {
			t.Fatalf("line = %q", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no syslog line arrived")
	}
}

func TestShipsBulkToElasticsearch(t *testing.T) {
	received := make(chan string, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path == "/_bulk" {
			received <- string(body)
		}
	}))
	defer server.Close()

	startShipping(t, dtos.LogDestination{Kind: enums.LogDestinationElasticsearch, Endpoint: server.URL})

	select {
	case body := <-received:
		if !strings.Contains(body, `"unit":"valkey-server"`) || !strings.Contains(body, `"message":"hello"`) || strings.Contains(body, "old line") {
			t.Fatalf("body = %s", body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no bulk request arrived")
	}
}

func TestSaveNullRemovesTheFile(t *testing.T) {
	home := t.TempDir()
	destination := dtos.LogDestination{Kind: enums.LogDestinationRsyslog, Endpoint: "tcp://10.0.0.1:514"}
	if err := Save(home, &destination); err != nil {
		t.Fatal(err)
	}
	if _, found := Load(home); !found {
		t.Fatal("destination not saved")
	}
	if err := Save(home, nil); err != nil {
		t.Fatal(err)
	}
	if _, found := Load(home); found {
		t.Fatal("destination not removed")
	}
}

func TestSaveRefusesAMismatchedScheme(t *testing.T) {
	err := Save(t.TempDir(), &dtos.LogDestination{Kind: enums.LogDestinationOpensearch, Endpoint: "tcp://10.0.0.1:9200"})

	if !errors.Is(err, ErrScheme) {
		t.Fatalf("err = %v", err)
	}
}
