package logship

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databasifylabs/engine-sdk/dtos"
	"github.com/databasifylabs/engine-sdk/enums"
)

const (
	batchLines   = 500
	batchWait    = 5 * time.Second
	scanInterval = time.Second
	maxBackoff   = 60 * time.Second
	maxBuffered  = 10000
	defaultIndex = "databasify-logs"
)

type Line struct {
	Unit    string
	Message string
	Time    time.Time
}

type sink interface {
	send(ctx context.Context, lines []Line) error
}

func ConfigPath(home string) string {
	return filepath.Join(home, "conf", "log-forwarding.json")
}

func Load(home string) (dtos.LogDestination, bool) {
	data, err := os.ReadFile(ConfigPath(home))
	if err != nil {
		return dtos.LogDestination{}, false
	}
	var destination dtos.LogDestination
	if json.Unmarshal(data, &destination) != nil {
		return dtos.LogDestination{}, false
	}

	return destination, true
}

func Run(ctx context.Context, destination dtos.LogDestination, logDir string) {
	target := sinkFor(destination)
	lines := make(chan Line, batchLines)
	go follow(ctx, logDir, lines)
	var buffered []Line
	backoff := time.Second
	flush := time.NewTicker(batchWait)
	defer flush.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-lines:
			buffered = append(buffered, line)
			if len(buffered) < batchLines {
				continue
			}
		case <-flush.C:
		}
		if len(buffered) == 0 {
			continue
		}
		if err := target.send(ctx, buffered); err != nil {
			buffered = buffered[max(0, len(buffered)-maxBuffered):]
			sleep(ctx, backoff)
			backoff = min(backoff*2, maxBackoff)

			continue
		}
		buffered, backoff = nil, time.Second
	}
}

func sleep(ctx context.Context, wait time.Duration) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func follow(ctx context.Context, logDir string, lines chan<- Line) {
	offsets := map[string]int64{}
	partial := map[string]string{}
	first := true
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()
	for {
		files, _ := filepath.Glob(filepath.Join(logDir, "*.log"))
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			offset, seen := offsets[file]
			if !seen && first {
				offsets[file] = info.Size()

				continue
			}
			if info.Size() < offset {
				offset, partial[file] = 0, ""
			}
			if info.Size() == offset {
				continue
			}
			chunk, next := readFrom(file, offset)
			offsets[file] = next
			unit := strings.TrimSuffix(filepath.Base(file), ".log")
			text := partial[file] + chunk
			complete := strings.Split(text, "\n")
			partial[file] = complete[len(complete)-1]
			for _, message := range complete[:len(complete)-1] {
				if message == "" {
					continue
				}
				select {
				case lines <- Line{Unit: unit, Message: message, Time: time.Now().UTC()}:
				case <-ctx.Done():
					return
				}
			}
		}
		first = false
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func readFrom(file string, offset int64) (string, int64) {
	handle, err := os.Open(file)
	if err != nil {
		return "", offset
	}
	defer handle.Close()
	if _, err := handle.Seek(offset, io.SeekStart); err != nil {
		return "", offset
	}
	data, _ := io.ReadAll(handle)

	return string(data), offset + int64(len(data))
}

func sinkFor(destination dtos.LogDestination) sink {
	if destination.Kind == enums.LogDestinationRsyslog {
		return &syslogSink{destination: destination}
	}

	return searchSink{destination: destination, client: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !destination.VerifyTls}}}}
}

type syslogSink struct {
	destination dtos.LogDestination
	conn        net.Conn
}

func (s *syslogSink) send(ctx context.Context, lines []Line) error {
	if s.conn == nil {
		conn, err := s.dial(ctx)
		if err != nil {
			return err
		}
		s.conn = conn
	}
	var frames bytes.Buffer
	for _, line := range lines {
		fmt.Fprintf(&frames, "<14>1 %s - %s - - - %s\n", line.Time.Format(time.RFC3339), line.Unit, line.Message)
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if _, err := s.conn.Write(frames.Bytes()); err != nil {
		_ = s.conn.Close()
		s.conn = nil

		return err
	}

	return nil
}

func (s *syslogSink) dial(ctx context.Context) (net.Conn, error) {
	endpoint, err := url.Parse(s.destination.Endpoint)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if endpoint.Scheme == "tls" {
		return (&tls.Dialer{NetDialer: dialer, Config: &tls.Config{InsecureSkipVerify: !s.destination.VerifyTls, ServerName: endpoint.Hostname()}}).DialContext(ctx, "tcp", endpoint.Host)
	}
	if endpoint.Scheme != "tcp" {
		return nil, errors.New("rsyslog endpoints use tcp:// or tls://")
	}

	return dialer.DialContext(ctx, "tcp", endpoint.Host)
}

type searchSink struct {
	destination dtos.LogDestination
	client      *http.Client
}

func (s searchSink) send(ctx context.Context, lines []Line) error {
	index := defaultIndex
	if s.destination.Index != nil {
		index = *s.destination.Index
	}
	var body bytes.Buffer
	action, _ := json.Marshal(map[string]map[string]string{"index": {"_index": index}})
	for _, line := range lines {
		document, _ := json.Marshal(map[string]string{"@timestamp": line.Time.Format(time.RFC3339Nano), "unit": line.Unit, "message": line.Message})
		body.Write(action)
		body.WriteByte('\n')
		body.Write(document)
		body.WriteByte('\n')
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(s.destination.Endpoint, "/")+"/_bulk", &body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-ndjson")
	if s.destination.Username != nil && s.destination.Password != nil {
		request.SetBasicAuth(*s.destination.Username, *s.destination.Password)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return fmt.Errorf("bulk request answered %s", response.Status)
	}

	return nil
}

func Save(home string, destination *dtos.LogDestination) error {
	path := ConfigPath(home)
	if destination == nil {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		return nil
	}
	if err := schemeFits(*destination); err != nil {
		return err
	}
	data, err := json.Marshal(destination)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func schemeFits(destination dtos.LogDestination) error {
	scheme, _, _ := strings.Cut(destination.Endpoint, "://")
	rsyslog := destination.Kind == enums.LogDestinationRsyslog
	if rsyslog && scheme != "https" || !rsyslog && scheme == "https" {
		return nil
	}

	return ErrScheme
}

var ErrScheme = errors.New("rsyslog endpoints use tcp:// or tls://, Elasticsearch and OpenSearch use https://")
