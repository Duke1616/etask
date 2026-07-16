package codebookcmd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/etask/internal/domain"
	codebooksvc "github.com/Duke1616/etask/internal/service/codebook"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestSystemOperationContext(t *testing.T) {
	ctx := systemOperationContext(context.Background())

	require.Equal(t, ctxutil.SystemTenantID, ctxutil.GetTenantID(ctx).Int64())
	require.Equal(t, systemCodebookAuthorUserID, ctxutil.GetUserID(ctx).Int64())
}

func TestLoadArtifactStorage(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("artifact.storage.driver", "local")
	viper.Set("artifact.storage.local.root", filepath.Join(t.TempDir(), "artifacts"))

	cfg, store, err := loadArtifactStorage()

	require.NoError(t, err)
	require.Equal(t, "local", cfg.Storage.Driver)
	require.NotNil(t, store)
}

func TestLoadArtifactStorageRejectsMissingDriver(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	_, _, err := loadArtifactStorage()

	require.ErrorContains(t, err, "初始化制品仓库存储失败")
}

func TestPrintImportResult(t *testing.T) {
	var output bytes.Buffer
	err := printImportResult(&output, codebooksvc.SystemImportResult{
		RootID: 57, RootName: "third_party", SourceDir: "/workspace/third_party",
		Directories: 6, Files: 17, Created: 23, Replaced: true,
	}, domain.ArtifactRelease{ID: 1, Digest: "abc123"})

	require.NoError(t, err)
	require.Equal(t, `SYSTEM 组件库导入并发布成功

导入信息
  根目录：third_party
  根节点 ID：57
  来源目录：/workspace/third_party

导入统计
  目录：6
  文件：17
  新建节点：23
  更新文件：0
  未变化节点：0
  跳过条目：0
  替换旧目录：是

制品信息
  发布 ID：1
  摘要：abc123
`, output.String())
}
