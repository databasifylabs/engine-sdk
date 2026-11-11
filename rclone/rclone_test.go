package rclone

import (
	"slices"
	"strings"
	"testing"

	"github.com/databasifylabs/engine-sdk/dtos"
)

func TestTargetKeepsKeysOutOfTheRemote(t *testing.T) {
	storage := dtos.Storage{Endpoint: "https://10.0.0.2:9000", Bucket: "backups", Region: "us-east-1", AccessKey: "key", SecretKey: "secret", PathStyle: true, Path: "/db1/"}

	target, env, flags := Target(storage, "base/2026.xbstream")

	if target != `:s3,provider=Other,endpoint="https://10.0.0.2:9000",region="us-east-1",force_path_style=true,no_check_bucket=true:backups/db1/base/2026.xbstream` {
		t.Fatalf("target = %s", target)
	}
	if strings.Contains(target, "secret") || !slices.Contains(env, "RCLONE_S3_SECRET_ACCESS_KEY=secret") || !slices.Equal(flags, []string{"--no-check-certificate"}) {
		t.Fatalf("env = %v, flags = %v", env, flags)
	}
}
