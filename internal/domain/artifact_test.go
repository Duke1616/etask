package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactRefValidate(t *testing.T) {
	validSystem := artifactRefForTest(CodebookScopeSystem, 0, "")
	validTenant := artifactRefForTest(CodebookScopeTenant, 7, "ops_common")
	testCases := []struct {
		name      string
		before    func(ref *ArtifactRef)
		reference ArtifactRef
		wantError string
	}{
		{name: "合法 SYSTEM 引用", reference: validSystem},
		{name: "合法租户制品引用", reference: validTenant},
		{name: "SYSTEM 不能指定项目", reference: validSystem, before: func(ref *ArtifactRef) { ref.ProjectID = 1 }, wantError: "项目 ID 必须为 0"},
		{name: "SYSTEM 不能指定命名空间", reference: validSystem, before: func(ref *ArtifactRef) { ref.Namespace = "common" }, wantError: "不能指定导入命名空间"},
		{name: "租户制品必须指定项目", reference: validTenant, before: func(ref *ArtifactRef) { ref.ProjectID = 0 }, wantError: "必须指定项目"},
		{name: "租户制品命名空间必须合法", reference: validTenant, before: func(ref *ArtifactRef) { ref.Namespace = "Bad-Name" }, wantError: "命名空间非法"},
		{name: "租户制品不能使用保留名", reference: validTenant, before: func(ref *ArtifactRef) { ref.Namespace = "etask" }, wantError: "命名空间非法"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reference := tc.reference
			if tc.before != nil {
				tc.before(&reference)
			}
			err := reference.Validate()
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateArtifactRefs(t *testing.T) {
	system := artifactRefForTest(CodebookScopeSystem, 0, "")
	first := artifactRefForTest(CodebookScopeTenant, 7, "ops_common")
	second := artifactRefForTest(CodebookScopeTenant, 9, "db_common")
	testCases := []struct {
		name      string
		refs      []ArtifactRef
		wantError string
	}{
		{name: "允许一个 SYSTEM 和多个租户项目", refs: []ArtifactRef{system, first, second}},
		{name: "拒绝重复 SYSTEM", refs: []ArtifactRef{system, system}, wantError: "重复的 SYSTEM"},
		{name: "拒绝重复租户项目", refs: []ArtifactRef{first, first}, wantError: "重复的租户项目"},
		{name: "拒绝重复命名空间", refs: []ArtifactRef{first, func() ArtifactRef { value := second; value.Namespace = first.Namespace; return value }()}, wantError: "重复命名空间"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateArtifactRefs(tc.refs)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func artifactRefForTest(scope CodebookScope, projectID int64, namespace string) ArtifactRef {
	return ArtifactRef{
		ReleaseID: 1, Digest: strings.Repeat("a", 64), BlobChecksum: strings.Repeat("b", 64),
		Size: 1, Format: "tar.zst", FormatVersion: 1,
		Scope: scope, ProjectID: projectID, Namespace: namespace,
	}
}
