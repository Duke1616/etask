package task

import (
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	codebookmocks "github.com/Duke1616/etask/internal/service/codebook/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSourceProjectID(t *testing.T) {
	testCases := []struct {
		name      string
		task      domain.Task
		prepare   func(*codebookmocks.MockService)
		projectID int64
		wantError string
	}{
		{
			name: "从 Codebook 绑定解析来源项目",
			task: domain.Task{
				GrpcConfig: &domain.GrpcConfig{Params: map[string]string{"code": "11"}},
				Metadata:   map[string]string{"code": "codebook"},
			},
			prepare: func(service *codebookmocks.MockService) {
				service.EXPECT().GetByID(gomock.Any(), int64(11)).Return(domain.Codebook{ID: 11, ProjectID: 9}, nil)
			},
			projectID: 9,
		},
		{
			name: "非 Codebook 绑定无需排除项目",
			task: domain.Task{
				GrpcConfig: &domain.GrpcConfig{Params: map[string]string{"variables": "22"}},
				Metadata:   map[string]string{"variables": "runner"},
			},
		},
		{
			name: "拒绝非法 Codebook 绑定",
			task: domain.Task{
				GrpcConfig: &domain.GrpcConfig{Params: map[string]string{"code": "invalid"}},
				Metadata:   map[string]string{"code": "codebook"},
			},
			wantError: "绑定 ID 非法",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			codebooks := codebookmocks.NewMockService(controller)
			if testCase.prepare != nil {
				testCase.prepare(codebooks)
			}
			service := &executionService{codebookSvc: codebooks}
			projectID, err := service.sourceProjectID(t.Context(), testCase.task)
			if testCase.wantError != "" {
				require.ErrorContains(t, err, testCase.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.projectID, projectID)
		})
	}
}
