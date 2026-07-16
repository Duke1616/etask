package codebookcmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Duke1616/etask/internal/domain"
	codebooksvc "github.com/Duke1616/etask/internal/service/codebook"
	"github.com/spf13/cobra"
)

// NewCommand 返回 codebook 运维子命令。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codebook",
		Short: "代码资源运维命令",
	}
	cmd.AddCommand(newImportSystemCommand())
	return cmd
}

type importSystemOptions struct {
	dir      string
	rootName string
	replace  bool
}

func newImportSystemCommand() *cobra.Command {
	opts := importSystemOptions{}
	cmd := &cobra.Command{
		Use:   "import-system",
		Short: "导入 SYSTEM 代码组件库并发布制品",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return opts.run(cmd.Context(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.dir, "dir", "", "需要导入的本地目录")
	cmd.Flags().StringVar(&opts.rootName, "root-name", "", "导入后的根目录名称，默认使用本地目录名")
	cmd.Flags().BoolVar(&opts.replace, "replace", false, "导入前删除同名 SYSTEM 根目录")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}

func (opts importSystemOptions) run(ctx context.Context, output io.Writer) error {
	plan, err := codebooksvc.PrepareSystemImport(codebooksvc.SystemImportRequest{
		SourceDir: opts.dir,
		RootName:  opts.rootName,
		Replace:   opts.replace,
	})
	if err != nil {
		return err
	}
	runtime, err := newRuntime()
	if err != nil {
		return err
	}
	result, release, err := runtime.importSystem(ctx, plan)
	if err != nil {
		return err
	}

	return printImportResult(output, result, release)
}

func printImportResult(output io.Writer, result codebooksvc.SystemImportResult,
	release domain.ArtifactRelease) error {
	var content strings.Builder
	fmt.Fprintln(&content, "SYSTEM 组件库导入并发布成功")
	fmt.Fprintln(&content)
	fmt.Fprintln(&content, "导入信息")
	fmt.Fprintf(&content, "  根目录：%s\n", result.RootName)
	fmt.Fprintf(&content, "  根节点 ID：%d\n", result.RootID)
	fmt.Fprintf(&content, "  来源目录：%s\n", result.SourceDir)
	fmt.Fprintln(&content)
	fmt.Fprintln(&content, "导入统计")
	fmt.Fprintf(&content, "  目录：%d\n", result.Directories)
	fmt.Fprintf(&content, "  文件：%d\n", result.Files)
	fmt.Fprintf(&content, "  新建节点：%d\n", result.Created)
	fmt.Fprintf(&content, "  更新文件：%d\n", result.Updated)
	fmt.Fprintf(&content, "  未变化节点：%d\n", result.Unchanged)
	fmt.Fprintf(&content, "  跳过条目：%d\n", result.Skipped)
	fmt.Fprintf(&content, "  替换旧目录：%s\n", chineseBool(result.Replaced))
	fmt.Fprintln(&content)
	fmt.Fprintln(&content, "制品信息")
	fmt.Fprintf(&content, "  发布 ID：%d\n", release.ID)
	fmt.Fprintf(&content, "  摘要：%s\n", release.Digest)
	_, err := io.WriteString(output, content.String())
	return err
}

func chineseBool(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
