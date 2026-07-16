package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodebookProjectValidateArtifactNamespace(t *testing.T) {
	project := CodebookProject{Name: "公共运维库", ArtifactEnabled: true, ArtifactNamespace: "ops_common"}
	require.NoError(t, project.Validate())

	tests := []struct {
		name      string
		namespace string
		message   string
	}{
		{name: "missing", message: "不能为空"},
		{name: "uppercase", namespace: "OpsCommon", message: "只能包含"},
		{name: "starts with number", namespace: "1ops", message: "只能包含"},
		{name: "system reserved", namespace: "etask", message: "保留命名空间"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := CodebookProject{Name: "公共库", ArtifactEnabled: true, ArtifactNamespace: tt.namespace}
			require.ErrorContains(t, value.Validate(), tt.message)
		})
	}
}
