package executor

import "testing"

func TestExecutorStartKey(t *testing.T) {
	testCases := []struct {
		name   string
		cursor string
		want   string
	}{
		{
			name: "empty cursor starts from index prefix",
			want: "/grpc/executor/",
		},
		{
			name:   "cursor skips previous group",
			cursor: "aliyun",
			want:   "/grpc/executor/aliyun0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := executorStartKey(tc.cursor); got != tc.want {
				t.Fatalf("executorStartKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseGroupName(t *testing.T) {
	testCases := []struct {
		name string
		key  string
		want string
		ok   bool
	}{
		{
			name: "valid",
			key:  "/grpc/executor/aliyun/127.0.0.1:9000",
			want: "aliyun",
			ok:   true,
		},
		{
			name: "invalid prefix",
			key:  "/grpc/services/aliyun/127.0.0.1:9000",
			ok:   false,
		},
		{
			name: "missing name",
			key:  "/grpc/executor/",
			ok:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseGroupName(tc.key)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("name = %q, want %q", got, tc.want)
			}
		})
	}
}
