package executor

import (
	"testing"

	"github.com/Duke1616/etask/internal/domain"
)

func TestMatchKeyword(t *testing.T) {
	exec := domain.Executor{
		Name: "aliyun",
		Desc: "cloud executor",
		Mode: "PUSH",
		Handlers: []domain.ExecutorHandler{
			{Name: "shell", Desc: "run command"},
		},
	}

	testCases := []struct {
		name    string
		keyword string
		want    bool
	}{
		{name: "empty", want: true},
		{name: "name", keyword: "ALI", want: true},
		{name: "desc", keyword: "cloud", want: true},
		{name: "mode", keyword: "push", want: true},
		{name: "handler name", keyword: "shell", want: true},
		{name: "handler desc", keyword: "command", want: true},
		{name: "miss", keyword: "ticket", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchKeyword(exec, tc.keyword); got != tc.want {
				t.Fatalf("matchKeyword() = %v, want %v", got, tc.want)
			}
		})
	}
}
