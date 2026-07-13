package pool

import (
	"testing"

	"github.com/Duke1616/etask/internal/domain"
)

func TestBindingMatcherAllow(t *testing.T) {
	matcher := NewBindingMatcher([]domain.ExecutionPoolBinding{
		{
			PoolName:    "default",
			HandlerName: "",
			Status:      domain.ExecutionPoolBindingStatusEnabled,
		},
		{
			PoolName:    "default",
			HandlerName: "blocked",
			Status:      domain.ExecutionPoolBindingStatusDisabled,
		},
		{
			PoolName:    "exact",
			HandlerName: "run",
			Status:      domain.ExecutionPoolBindingStatusEnabled,
		},
	})

	testCases := []struct {
		name    string
		pool    string
		handler string
		want    bool
	}{
		{name: "wildcard allows handler", pool: "default", handler: "run", want: true},
		{name: "exact disabled overrides wildcard", pool: "default", handler: "blocked", want: false},
		{name: "exact allows handler", pool: "exact", handler: "run", want: true},
		{name: "missing wildcard denies", pool: "exact", handler: "shell", want: false},
		{name: "missing pool denies", pool: "missing", handler: "run", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matcher.Allow(tc.pool, tc.handler); got != tc.want {
				t.Fatalf("Allow() = %t, want %t", got, tc.want)
			}
		})
	}
}
