package repository

import (
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository/dao"
	"github.com/stretchr/testify/require"
)

func TestArtifactSnapshotFiles(t *testing.T) {
	nodes := []dao.Codebook{
		{ID: 1, Kind: domain.CodebookKindDirectory.String(), Name: "private"},
		{ID: 2, ParentID: 1, Kind: domain.CodebookKindFile.String(), Name: "util.py", CurrentVersionID: 10},
	}
	snapshot := newArtifactSnapshot(nodes, []dao.CodebookVersion{
		{ID: 10, Hash: "hash", Code: "VALUE = 1"},
	})

	files, err := snapshot.Files()
	require.NoError(t, err)
	require.Equal(t, []domain.ArtifactFile{{Path: "private/util.py", Hash: "hash", Code: "VALUE = 1"}}, files)
}

func TestArtifactSnapshotRejectsBrokenTree(t *testing.T) {
	tests := []struct {
		name  string
		nodes []dao.Codebook
		err   string
	}{
		{
			name: "父节点不存在",
			nodes: []dao.Codebook{
				{ID: 1, ParentID: 99, Kind: domain.CodebookKindFile.String(), Name: "util.py", CurrentVersionID: 10},
			},
			err: "父节点 99 不存在",
		},
		{
			name: "目录循环引用",
			nodes: []dao.Codebook{
				{ID: 1, ParentID: 2, Kind: domain.CodebookKindDirectory.String(), Name: "a"},
				{ID: 2, ParentID: 1, Kind: domain.CodebookKindDirectory.String(), Name: "b"},
				{ID: 3, ParentID: 1, Kind: domain.CodebookKindFile.String(), Name: "util.py", CurrentVersionID: 10},
			},
			err: "循环引用",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := newArtifactSnapshot(tt.nodes, []dao.CodebookVersion{{ID: 10}})
			_, err := snapshot.Files()
			require.ErrorContains(t, err, tt.err)
		})
	}
}

func TestArtifactCurrentVersionIDs(t *testing.T) {
	_, err := artifactCurrentVersionIDs([]dao.Codebook{
		{ID: 1, Kind: domain.CodebookKindFile.String(), Name: "util.py"},
	})
	require.ErrorContains(t, err, "尚未设置当前版本")
}
