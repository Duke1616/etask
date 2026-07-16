package preview

import (
	"context"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	codebookmocks "github.com/Duke1616/etask/internal/service/codebook/mocks"
	runnermocks "github.com/Duke1616/etask/internal/service/runner/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPrepareMergesTemporaryVariables(t *testing.T) {
	ctrl := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(ctrl)
	runners := runnermocks.NewMockService(ctrl)
	codebooks.EXPECT().GetByID(gomock.Any(), int64(11)).Return(domain.Codebook{
		ID: 11, Name: "main.py", Kind: domain.CodebookKindFile,
	}, nil)
	runners.EXPECT().FindByID(gomock.Any(), int64(22)).Return(domain.Runner{
		ID: 22, CodebookID: 11, Kind: domain.RunnerKindGRPC, Target: "executor",
		Handler: "python", Action: domain.RunnerActionRegistered,
	}, nil)
	runners.EXPECT().ListMergedVariables(gomock.Any(), int64(22)).Return([]domain.RunnerVariable{
		{Key: "REGION", Value: "default"},
		{Key: "TOKEN", Value: "secret", Secret: true},
	}, nil)

	svc := &service{codebookSvc: codebooks, runnerSvc: runners}
	result, err := svc.prepare(context.Background(), RunCommand{
		CodebookID: 11,
		RunnerID:   22,
		Code:       "print('ok')",
		Variables: []domain.RunnerVariable{
			{Key: "REGION", Value: "temporary"},
			{Key: "DEBUG", Value: "true"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "{}", result.args)
	require.Equal(t, defaultTimeoutSeconds, result.timeout)
	require.Equal(t, []previewVariable{
		{Key: "REGION", Value: "temporary"},
		{Key: "TOKEN", Value: "secret", Secret: true},
		{Key: "DEBUG", Value: "true"},
	}, result.variables)
}

func TestPrepareRejectsRunnerFromAnotherCodebook(t *testing.T) {
	ctrl := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(ctrl)
	runners := runnermocks.NewMockService(ctrl)
	codebooks.EXPECT().GetByID(gomock.Any(), int64(11)).Return(domain.Codebook{
		ID: 11, Name: "main.py", Kind: domain.CodebookKindFile,
	}, nil)
	runners.EXPECT().FindByID(gomock.Any(), int64(22)).Return(domain.Runner{
		ID: 22, CodebookID: 99, Kind: domain.RunnerKindGRPC, Target: "executor",
		Handler: "python", Action: domain.RunnerActionRegistered,
	}, nil)

	svc := &service{codebookSvc: codebooks, runnerSvc: runners}
	_, err := svc.prepare(context.Background(), RunCommand{
		CodebookID: 11, RunnerID: 22, Code: "print('ok')",
	})

	require.ErrorContains(t, err, "未绑定当前 Codebook")
}

func TestPrepareAcceptsKafkaRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	codebooks := codebookmocks.NewMockService(ctrl)
	runners := runnermocks.NewMockService(ctrl)
	codebooks.EXPECT().GetByID(gomock.Any(), int64(11)).Return(domain.Codebook{
		ID: 11, Name: "main.sh", Kind: domain.CodebookKindFile,
	}, nil)
	runners.EXPECT().FindByID(gomock.Any(), int64(22)).Return(domain.Runner{
		ID: 22, CodebookID: 11, Kind: domain.RunnerKindKafka, Target: "agent-shell",
		Handler: "shell", Action: domain.RunnerActionRegistered,
	}, nil)
	runners.EXPECT().ListMergedVariables(gomock.Any(), int64(22)).Return(nil, nil)

	svc := &service{codebookSvc: codebooks, runnerSvc: runners}
	result, err := svc.prepare(context.Background(), RunCommand{
		CodebookID: 11, RunnerID: 22, Code: "echo ok",
	})

	require.NoError(t, err)
	require.Equal(t, domain.RunnerKindKafka, result.runner.Kind)
	require.Equal(t, "agent-shell", result.runner.Target)
}
