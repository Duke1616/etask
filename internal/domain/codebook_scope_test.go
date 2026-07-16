package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodebookScopeValidateWriteAccess(t *testing.T) {
	const systemTenantID int64 = 1

	require.NoError(t, CodebookScopeTenant.ValidateWriteAccess(10, systemTenantID))
	require.NoError(t, CodebookScopeSystem.ValidateWriteAccess(systemTenantID, systemTenantID))
	require.ErrorContains(t,
		CodebookScopeSystem.ValidateWriteAccess(10, systemTenantID),
		"只有系统租户",
	)
	require.ErrorContains(t,
		CodebookScopeTenant.ValidateWriteAccess(0, systemTenantID),
		"缺少租户上下文",
	)
}

func TestCodebookVersionAllowsEmptyFile(t *testing.T) {
	version := CodebookVersion{NodeID: 1}
	err := version.PrepareForNode(Codebook{ID: 1, Kind: CodebookKindFile, Scope: CodebookScopeTenant})
	require.NoError(t, err)
}
