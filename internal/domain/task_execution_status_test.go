package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNonTerminalTaskExecutionStatuses(t *testing.T) {
	statuses := NonTerminalTaskExecutionStatuses()
	require.ElementsMatch(t, []TaskExecutionStatus{
		TaskExecutionStatusWaitingPull,
		TaskExecutionStatusPrepare,
		TaskExecutionStatusRunning,
		TaskExecutionStatusFailedRetryable,
		TaskExecutionStatusFailedRescheduled,
	}, statuses)
	require.NotContains(t, statuses, TaskExecutionStatusSuccess)
	require.NotContains(t, statuses, TaskExecutionStatusFailed)

	statuses[0] = TaskExecutionStatusSuccess
	require.NotContains(t, NonTerminalTaskExecutionStatuses(), TaskExecutionStatusSuccess)
}
