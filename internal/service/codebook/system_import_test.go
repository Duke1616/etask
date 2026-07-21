package codebook_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Duke1616/etask/internal/domain"
	codebooksvc "github.com/Duke1616/etask/internal/service/codebook"
	codebookmocks "github.com/Duke1616/etask/internal/service/codebook/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSystemImporterCreatesTree(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "config"), []byte("ignored"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".DS_Store"), []byte("ignored"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(root, "__pycache__"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "__pycache__", "main.pyc"), []byte{0xff}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "module.pyo"), []byte("ignored"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "binary.dat"), []byte{0xff, 0xfe}, 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(root, "private"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "private", "helper.py"), []byte("helper"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.py"), []byte("main"), 0o644))

	ctrl := gomock.NewController(t)
	svc := codebookmocks.NewMockService(ctrl)
	ctx := context.Background()
	gomock.InOrder(
		svc.EXPECT().Children(ctx, int64(0), int64(0)).Return(nil, nil),
		svc.EXPECT().Create(ctx, domain.Codebook{
			Scope: domain.CodebookScopeSystem, Name: "library", Kind: domain.CodebookKindDirectory,
		}).Return(int64(1), nil),
		svc.EXPECT().Create(ctx, domain.Codebook{
			Scope: domain.CodebookScopeSystem, ParentID: 1, Name: "main.py",
			Kind: domain.CodebookKindFile, Code: "main",
		}).Return(int64(2), nil),
		svc.EXPECT().Create(ctx, domain.Codebook{
			Scope: domain.CodebookScopeSystem, ParentID: 1, Name: "private", Kind: domain.CodebookKindDirectory,
		}).Return(int64(3), nil),
		svc.EXPECT().Create(ctx, domain.Codebook{
			Scope: domain.CodebookScopeSystem, ParentID: 3, Name: "helper.py",
			Kind: domain.CodebookKindFile, Code: "helper",
		}).Return(int64(4), nil),
	)

	plan, err := codebooksvc.PrepareSystemImport(codebooksvc.SystemImportRequest{
		SourceDir: root,
		RootName:  "library",
	})
	require.NoError(t, err)
	result, err := codebooksvc.NewSystemImporter(svc).Import(ctx, plan)

	require.NoError(t, err)
	require.Equal(t, int64(1), result.RootID)
	require.Equal(t, 2, result.Directories)
	require.Equal(t, 2, result.Files)
	require.Equal(t, 4, result.Created)
	require.Equal(t, 0, result.Updated)
	require.Equal(t, 0, result.Unchanged)
	require.Equal(t, 5, result.Skipped)
}

func TestSystemImporterCreatesVersionForChangedFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.py"), []byte("new"), 0o644))

	ctrl := gomock.NewController(t)
	svc := codebookmocks.NewMockService(ctrl)
	ctx := context.Background()
	gomock.InOrder(
		svc.EXPECT().Children(ctx, int64(0), int64(0)).Return([]domain.Codebook{{
			ID: 1, Scope: domain.CodebookScopeSystem, Name: "library", Kind: domain.CodebookKindDirectory,
		}}, nil),
		svc.EXPECT().Children(ctx, int64(0), int64(1)).Return([]domain.Codebook{{
			ID: 2, Scope: domain.CodebookScopeSystem, ParentID: 1, Name: "main.py", Kind: domain.CodebookKindFile,
		}}, nil),
		svc.EXPECT().GetByID(ctx, int64(2)).Return(domain.Codebook{
			ID: 2, Scope: domain.CodebookScopeSystem, ParentID: 1, Name: "main.py",
			Kind: domain.CodebookKindFile, Code: "old",
		}, nil),
		svc.EXPECT().CreateVersion(ctx, domain.CodebookVersionCreate{
			NodeID: 2, Code: "new", Message: "SYSTEM 组件库增量导入",
		}).Return(int64(7), nil),
		svc.EXPECT().UseVersion(ctx, int64(2), int64(7)).Return(int64(1), nil),
	)

	plan, err := codebooksvc.PrepareSystemImport(codebooksvc.SystemImportRequest{
		SourceDir: root,
		RootName:  "library",
	})
	require.NoError(t, err)
	result, err := codebooksvc.NewSystemImporter(svc).Import(ctx, plan)

	require.NoError(t, err)
	require.Equal(t, 1, result.Directories)
	require.Equal(t, 1, result.Files)
	require.Equal(t, 0, result.Created)
	require.Equal(t, 1, result.Updated)
	require.Equal(t, 1, result.Unchanged)
}

func TestSystemImporterReplacesExistingRoot(t *testing.T) {
	root := t.TempDir()

	ctrl := gomock.NewController(t)
	svc := codebookmocks.NewMockService(ctrl)
	ctx := context.Background()
	gomock.InOrder(
		svc.EXPECT().Children(ctx, int64(0), int64(0)).Return([]domain.Codebook{{
			ID: 1, Scope: domain.CodebookScopeSystem, Name: "library", Kind: domain.CodebookKindDirectory,
		}}, nil),
		svc.EXPECT().Delete(ctx, int64(1)).Return(int64(3), nil),
		svc.EXPECT().Create(ctx, domain.Codebook{
			Scope: domain.CodebookScopeSystem, Name: "library", Kind: domain.CodebookKindDirectory,
		}).Return(int64(8), nil),
	)

	plan, err := codebooksvc.PrepareSystemImport(codebooksvc.SystemImportRequest{
		SourceDir: root,
		RootName:  "library",
		Replace:   true,
	})
	require.NoError(t, err)
	result, err := codebooksvc.NewSystemImporter(svc).Import(ctx, plan)

	require.NoError(t, err)
	require.True(t, result.Replaced)
	require.Equal(t, int64(8), result.RootID)
	require.Equal(t, 1, result.Created)
	require.Equal(t, 0, result.Unchanged)
}
