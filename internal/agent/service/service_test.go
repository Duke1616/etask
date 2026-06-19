package service

import (
	"testing"

	"github.com/Duke1616/etask/internal/agent/domain"
)

func TestFilterAgents(t *testing.T) {
	svc := &service{}
	agents := []domain.Agent{
		{
			Name:  "agent-a",
			Desc:  "default agent",
			Topic: "topic-a",
			Handlers: []domain.HandlerDetail{
				{Name: "shell", Desc: "run command"},
			},
		},
		{
			Name:  "agent-b",
			Desc:  "backup",
			Topic: "topic-b",
			Handlers: []domain.HandlerDetail{
				{Name: "python", Desc: "run script"},
			},
		},
	}

	testCases := []struct {
		name    string
		keyword string
		want    int
	}{
		{name: "empty", want: 2},
		{name: "name", keyword: "AGENT-A", want: 1},
		{name: "topic", keyword: "topic-b", want: 1},
		{name: "handler name", keyword: "shell", want: 1},
		{name: "handler desc", keyword: "script", want: 1},
		{name: "miss", keyword: "aliyun", want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(svc.filterAgents(agents, tc.keyword)); got != tc.want {
				t.Fatalf("len(filterAgents()) = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPageAgents(t *testing.T) {
	svc := &service{}
	agents := []domain.Agent{
		{Name: "c"},
		{Name: "a"},
		{Name: "b"},
	}

	first := svc.pageAgents(agents, 2, "")
	if len(first.Agents) != 2 {
		t.Fatalf("len(first.Agents) = %d, want 2", len(first.Agents))
	}
	if first.Agents[0].Name != "a" || first.Agents[1].Name != "b" {
		t.Fatalf("first agents = %#v, want a,b", first.Agents)
	}
	if first.NextCursor != "b" {
		t.Fatalf("first.NextCursor = %q, want b", first.NextCursor)
	}

	second := svc.pageAgents(agents, 2, first.NextCursor)
	if len(second.Agents) != 1 || second.Agents[0].Name != "c" {
		t.Fatalf("second agents = %#v, want c", second.Agents)
	}
	if second.NextCursor != "" {
		t.Fatalf("second.NextCursor = %q, want empty", second.NextCursor)
	}
}
