package rclone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/databasifylabs/engine-sdk/dtos"
)

const binary = "/usr/local/bin/rclone"

type Entry struct {
	Path    string    `json:"Path"`
	Size    int64     `json:"Size"`
	ModTime time.Time `json:"ModTime"`
	IsDir   bool      `json:"IsDir"`
}

func Target(storage dtos.Storage, path string) (string, []string, []string) {
	remote := fmt.Sprintf(`:s3,provider=Other,endpoint="%s",region="%s",force_path_style=%t,no_check_bucket=true:%s/%s`,
		storage.Endpoint, storage.Region, storage.PathStyle, storage.Bucket, strings.Trim(strings.Trim(storage.Path, "/")+"/"+strings.TrimLeft(path, "/"), "/"))
	env := []string{"RCLONE_S3_ACCESS_KEY_ID=" + storage.AccessKey, "RCLONE_S3_SECRET_ACCESS_KEY=" + storage.SecretKey, "RCLONE_CONFIG=/dev/null"}
	flags := []string{}
	if !storage.VerifyTls {
		flags = append(flags, "--no-check-certificate")
	}

	return remote, env, flags
}

func Rcat(ctx context.Context, storage dtos.Storage, input io.Reader, path string) error {
	_, err := run(ctx, storage, input, nil, "rcat", path)

	return err
}

func Cat(ctx context.Context, storage dtos.Storage, output io.Writer, path string) error {
	_, err := run(ctx, storage, nil, output, "cat", path)

	return err
}

func CopyTo(ctx context.Context, storage dtos.Storage, local string, path string) error {
	_, err := run(ctx, storage, nil, nil, "copyto", path, local)

	return err
}

func Fetch(ctx context.Context, storage dtos.Storage, path string, local string) error {
	target, env, flags := Target(storage, path)
	command := exec.CommandContext(ctx, binary, append(append([]string{"copy"}, flags...), target, local)...)
	command.Env = append(os.Environ(), env...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("rclone copy: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func List(ctx context.Context, storage dtos.Storage, path string) ([]Entry, error) {
	output, err := run(ctx, storage, nil, nil, "lsjson", path)
	if err != nil {
		if strings.Contains(err.Error(), "directory not found") {
			return []Entry{}, nil
		}

		return nil, err
	}
	entries := []Entry{}

	return entries, json.Unmarshal([]byte(output), &entries)
}

func Delete(ctx context.Context, storage dtos.Storage, path string) error {
	_, err := run(ctx, storage, nil, nil, "deletefile", path)

	return err
}

func run(ctx context.Context, storage dtos.Storage, input io.Reader, output io.Writer, verb string, path string, local ...string) (string, error) {
	target, env, flags := Target(storage, path)
	args := append([]string{verb}, flags...)
	args = append(append(args, local...), target)
	command := exec.CommandContext(ctx, binary, args...)
	command.Env = append(os.Environ(), env...)
	command.Stdin = input
	var stderr strings.Builder
	command.Stderr = &stderr
	var stdout strings.Builder
	if output != nil {
		command.Stdout = output
	} else {
		command.Stdout = &stdout
	}
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("rclone %s: %w: %s", verb, err, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}
