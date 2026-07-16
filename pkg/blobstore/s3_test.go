package blobstore

import (
	"net/http"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

func TestParseMinIOEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		defaultSecure bool
		endpoint      string
		secure        bool
	}{
		{name: "http", value: "http://minio:9000", endpoint: "minio:9000"},
		{name: "https", value: "https://s3.example.com/", endpoint: "s3.example.com", secure: true},
		{name: "without scheme", value: "minio:9000", endpoint: "minio:9000"},
		{name: "without scheme trailing slash", value: "minio:9000/", endpoint: "minio:9000"},
		{name: "secure without scheme", value: "s3.example.com", defaultSecure: true, endpoint: "s3.example.com", secure: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, secure, err := parseMinIOEndpoint(tt.value, tt.defaultSecure)
			require.NoError(t, err)
			require.Equal(t, tt.endpoint, endpoint)
			require.Equal(t, tt.secure, secure)
		})
	}
}

func TestNewS3ValidatesConfigWithoutConnecting(t *testing.T) {
	store, err := NewS3(S3Config{
		Endpoint:  "http://minio:9000",
		Bucket:    "etask-artifacts",
		Prefix:    "codebook/releases",
		AccessKey: "access-key",
		SecretKey: "secret-key",
	})
	require.NoError(t, err)
	require.Equal(t, "etask-artifacts", store.bucket)
	require.Equal(t, "codebook/releases", store.prefix)

	_, err = NewS3(S3Config{Endpoint: "minio:9000", Bucket: "INVALID_BUCKET"})
	require.ErrorContains(t, err, "存储桶名称非法")
	_, err = NewS3(S3Config{
		Endpoint: "minio:9000", Bucket: "etask-artifacts", Prefix: "../outside",
	})
	require.ErrorContains(t, err, "对象前缀非法")
	_, err = NewS3(S3Config{
		Endpoint: "minio:9000", Bucket: "etask-artifacts", AccessKey: "access-key",
	})
	require.ErrorContains(t, err, "必须同时配置")
}

func TestParseMinIOEndpointRejectsInvalidValue(t *testing.T) {
	for _, value := range []string{"", "ftp://minio:9000", "http://minio:9000/path", "minio:9000/path"} {
		t.Run(value, func(t *testing.T) {
			_, _, err := parseMinIOEndpoint(value, false)
			require.Error(t, err)
		})
	}
}

func TestS3ResolveAndNotFound(t *testing.T) {
	store := &S3{prefix: "codebook"}
	resolved, err := store.resolve("artifacts/release.tar.zst")
	require.NoError(t, err)
	require.Equal(t, "codebook/artifacts/release.tar.zst", resolved)
	_, err = store.resolve("../outside")
	require.Error(t, err)

	require.True(t, isMinIONotFound(minio.ErrorResponse{Code: "NoSuchKey", StatusCode: http.StatusNotFound}))
}
