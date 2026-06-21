package codebookcmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Duke1616/eiam/pkg/ctxutil"
	"github.com/Duke1616/eiam/pkg/gormx"
	"github.com/Duke1616/etask/internal/domain"
	"github.com/Duke1616/etask/internal/repository"
	"github.com/Duke1616/etask/internal/repository/dao"
	codebooksvc "github.com/Duke1616/etask/internal/service/codebook"
	"github.com/Duke1616/etask/ioc"
	"github.com/Duke1616/etask/pkg/sorter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const systemCodebookAuthorUserID int64 = 1

// NewCommand 返回 codebook 运维子命令。
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codebook",
		Short: "代码资源运维命令",
	}

	cmd.AddCommand(newImportSystemCommand())
	return cmd
}

func newImportSystemCommand() *cobra.Command {
	opts := importSystemOptions{}
	cmd := &cobra.Command{
		Use:   "import-system",
		Short: "将本地目录导入为 SYSTEM 代码组件库",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run(cmd.Context())
		},
	}
	cmd.Flags().StringVar(&opts.dir, "dir", "", "需要导入的本地目录")
	cmd.Flags().StringVar(&opts.rootName, "root-name", "", "导入后的根目录名称，默认使用本地目录名")
	cmd.Flags().BoolVar(&opts.replace, "replace", false, "导入前删除同名 SYSTEM 根目录")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}

type importSystemOptions struct {
	dir      string
	rootName string
	replace  bool
}

type importSystemStats struct {
	rootID    int64
	dirs      int
	files     int
	created   int
	updated   int
	unchanged int
	skipped   int
	replaced  bool
	rootName  string
	sourceDir string
}

type importSystemPlan struct {
	sourceDir string
	rootName  string
}

type systemDirectoryImporter struct {
	ctx                context.Context
	db                 *gorm.DB
	svc                codebooksvc.Service
	sourceDir          string
	rootName           string
	parentIDs          map[string]int64
	nextSortNoByParent map[int64]int64
	stats              importSystemStats
}

func (opts importSystemOptions) run(ctx context.Context) error {
	if strings.TrimSpace(viper.GetString("mysql.dsn")) == "" {
		return fmt.Errorf("mysql.dsn 不能为空")
	}
	ctx = ctxutil.WithUserID(ctx, systemCodebookAuthorUserID)
	plan, err := opts.prepare()
	if err != nil {
		return err
	}

	db := ioc.InitDB()
	stats, err := importSystemDirectory(ctx, db, plan, opts.replace)
	if err != nil {
		return err
	}

	fmt.Printf("SYSTEM 组件库导入完成: root_id=%d root_name=%s source=%s dirs=%d files=%d created=%d updated=%d unchanged=%d skipped=%d replaced=%t\n",
		stats.rootID, stats.rootName, stats.sourceDir, stats.dirs, stats.files,
		stats.created, stats.updated, stats.unchanged, stats.skipped, stats.replaced)
	return nil
}

func (opts importSystemOptions) prepare() (importSystemPlan, error) {
	sourceDir, err := filepath.Abs(strings.TrimSpace(opts.dir))
	if err != nil {
		return importSystemPlan{}, fmt.Errorf("解析导入目录失败: %w", err)
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		return importSystemPlan{}, fmt.Errorf("读取导入目录失败: %w", err)
	}
	if !info.IsDir() {
		return importSystemPlan{}, fmt.Errorf("导入路径必须是目录: %s", sourceDir)
	}

	rootName := strings.TrimSpace(opts.rootName)
	if rootName == "" {
		rootName = filepath.Base(sourceDir)
	}
	if rootName == "" || rootName == "." || rootName == string(os.PathSeparator) {
		return importSystemPlan{}, fmt.Errorf("根目录名称不能为空")
	}
	if strings.ContainsAny(rootName, `/\`) {
		return importSystemPlan{}, fmt.Errorf("根目录名称不能包含路径分隔符: %s", rootName)
	}
	return importSystemPlan{sourceDir: sourceDir, rootName: rootName}, nil
}

func newCodebookService(db *gorm.DB) codebooksvc.Service {
	codebookDAO := dao.NewGORMCodebookDAO(db)
	projectDAO := dao.NewGORMCodebookProjectDAO(db)
	repo := repository.NewCodebookRepository(codebookDAO, projectDAO)
	return codebooksvc.NewService(repo)
}

func importSystemDirectory(ctx context.Context, db *gorm.DB, plan importSystemPlan, replace bool) (importSystemStats, error) {
	var stats importSystemStats
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var replaced bool
		if replace {
			var err error
			replaced, err = deleteExistingSystemRoot(ctx, tx, plan.rootName)
			if err != nil {
				return err
			}
		}

		importer := newSystemDirectoryImporter(ctx, tx, newCodebookService(tx), plan)
		var err error
		stats, err = importer.Import()
		stats.replaced = replaced
		return err
	})
	return stats, err
}

func newSystemDirectoryImporter(ctx context.Context, db *gorm.DB, svc codebooksvc.Service, plan importSystemPlan) *systemDirectoryImporter {
	return &systemDirectoryImporter{
		ctx:                ctx,
		db:                 db,
		svc:                svc,
		sourceDir:          plan.sourceDir,
		rootName:           plan.rootName,
		parentIDs:          make(map[string]int64),
		nextSortNoByParent: make(map[int64]int64),
		stats: importSystemStats{
			rootName:  plan.rootName,
			sourceDir: plan.sourceDir,
		},
	}
}

func (i *systemDirectoryImporter) Import() (importSystemStats, error) {
	if err := i.createRoot(); err != nil {
		return i.stats, err
	}
	err := filepath.WalkDir(i.sourceDir, i.importEntry)
	return i.stats, err
}

func (i *systemDirectoryImporter) createRoot() error {
	rootID, created, err := i.createOrReuseDirectory(0, i.rootName)
	if err != nil {
		return fmt.Errorf("创建 SYSTEM 根目录失败: %w", err)
	}
	i.parentIDs[i.sourceDir] = rootID
	i.stats.rootID = rootID
	i.stats.dirs = 1
	i.countNodeResult(created)
	return nil
}

func (i *systemDirectoryImporter) importEntry(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return fmt.Errorf("读取路径失败 %s: %w", path, walkErr)
	}
	if path == i.sourceDir {
		return nil
	}
	if shouldSkipImportEntry(entry) {
		i.stats.skipped++
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if entry.IsDir() {
		return i.importDirectory(path, entry)
	}
	if !entry.Type().IsRegular() {
		i.stats.skipped++
		return nil
	}
	return i.importFile(path, entry)
}

func (i *systemDirectoryImporter) importDirectory(path string, entry fs.DirEntry) error {
	parentID, err := i.parentID(path)
	if err != nil {
		return err
	}
	id, created, err := i.createOrReuseDirectory(parentID, entry.Name())
	if err != nil {
		return fmt.Errorf("创建目录失败 %s: %w", path, err)
	}
	i.parentIDs[path] = id
	i.stats.dirs++
	i.countNodeResult(created)
	return nil
}

func (i *systemDirectoryImporter) importFile(path string, entry fs.DirEntry) error {
	parentID, err := i.parentID(path)
	if err != nil {
		return err
	}
	code, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败 %s: %w", path, err)
	}
	i.stats.files++

	old, ok, err := i.findNode(parentID, entry.Name())
	if err != nil {
		return err
	}
	if ok {
		return i.updateExistingFile(path, old, string(code))
	}

	node, err := i.nodeForPath(path, entry.Name())
	if err != nil {
		return err
	}
	node.Kind = domain.CodebookKindFile
	node.Code = string(code)
	if _, err = i.svc.Create(i.ctx, node); err != nil {
		return fmt.Errorf("创建文件失败 %s: %w", path, err)
	}
	i.stats.created++
	return nil
}

func (i *systemDirectoryImporter) nodeForPath(path, name string) (domain.Codebook, error) {
	parentID, err := i.parentID(path)
	if err != nil {
		return domain.Codebook{}, err
	}
	sortNo, err := i.nextChildSortNo(parentID)
	if err != nil {
		return domain.Codebook{}, err
	}
	return domain.Codebook{
		TenantID:  ctxutil.SystemTenantID,
		Scope:     domain.CodebookScopeSystem,
		ProjectID: 0,
		ParentID:  parentID,
		Name:      name,
		SortNo:    sortNo,
	}, nil
}

func (i *systemDirectoryImporter) parentID(path string) (int64, error) {
	parentID, ok := i.parentIDs[filepath.Dir(path)]
	if !ok {
		return 0, fmt.Errorf("父目录尚未导入: %s", filepath.Dir(path))
	}
	return parentID, nil
}

func (i *systemDirectoryImporter) createOrReuseDirectory(parentID int64, name string) (int64, bool, error) {
	old, ok, err := i.findNode(parentID, name)
	if err != nil {
		return 0, false, err
	}
	if ok {
		if old.Kind != domain.CodebookKindDirectory.String() {
			return 0, false, fmt.Errorf("同名节点已存在但不是目录: %s", name)
		}
		return old.ID, false, nil
	}

	sortNo, err := i.nextChildSortNo(parentID)
	if err != nil {
		return 0, false, err
	}
	id, err := i.svc.Create(i.ctx, domain.Codebook{
		TenantID:  ctxutil.SystemTenantID,
		Scope:     domain.CodebookScopeSystem,
		ProjectID: 0,
		ParentID:  parentID,
		Name:      name,
		Kind:      domain.CodebookKindDirectory,
		SortNo:    sortNo,
	})
	return id, true, err
}

func (i *systemDirectoryImporter) updateExistingFile(path string, old dao.Codebook, code string) error {
	if old.Kind != domain.CodebookKindFile.String() {
		return fmt.Errorf("同名节点已存在但不是文件: %s", path)
	}

	current, err := i.svc.GetByID(i.ctx, old.ID)
	if err != nil {
		return fmt.Errorf("读取已有文件失败 %s: %w", path, err)
	}
	if current.Code == code {
		i.stats.unchanged++
		return nil
	}
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("文件内容为空，无法为已有文件创建新版本: %s", path)
	}
	versionID, err := i.svc.CreateVersion(i.ctx, domain.CodebookVersion{
		NodeID:       old.ID,
		Code:         code,
		Message:      "SYSTEM 组件库增量导入",
		AuthorUserID: systemCodebookAuthorUserID,
	})
	if err != nil {
		return fmt.Errorf("创建文件新版本失败 %s: %w", path, err)
	}
	if _, err = i.svc.UseVersion(i.ctx, old.ID, versionID); err != nil {
		return fmt.Errorf("切换文件当前版本失败 %s: %w", path, err)
	}
	i.stats.updated++
	return nil
}

func (i *systemDirectoryImporter) findNode(parentID int64, name string) (dao.Codebook, bool, error) {
	ctx := gormx.IgnoreTenantContext(i.ctx)
	var node dao.Codebook
	err := i.db.WithContext(ctx).
		Where("tenant_id = ? AND scope = ? AND project_id = ? AND parent_id = ? AND name = ?",
			ctxutil.SystemTenantID, domain.CodebookScopeSystem.String(), 0, parentID, name).
		First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return dao.Codebook{}, false, nil
	}
	if err != nil {
		return dao.Codebook{}, false, fmt.Errorf("查询同名节点失败: %w", err)
	}
	return node, true, nil
}

func (i *systemDirectoryImporter) countNodeResult(created bool) {
	if created {
		i.stats.created++
		return
	}
	i.stats.unchanged++
}

func (i *systemDirectoryImporter) nextChildSortNo(parentID int64) (int64, error) {
	if _, ok := i.nextSortNoByParent[parentID]; !ok {
		maxSortNo, err := i.maxSortNo(parentID)
		if err != nil {
			return 0, err
		}
		i.nextSortNoByParent[parentID] = maxSortNo
	}
	i.nextSortNoByParent[parentID] += sorter.DefaultIndexGap
	return i.nextSortNoByParent[parentID], nil
}

func (i *systemDirectoryImporter) maxSortNo(parentID int64) (int64, error) {
	ctx := gormx.IgnoreTenantContext(i.ctx)
	var maxSortNo int64
	err := i.db.WithContext(ctx).
		Model(&dao.Codebook{}).
		Where("tenant_id = ? AND scope = ? AND project_id = ? AND parent_id = ?",
			ctxutil.SystemTenantID, domain.CodebookScopeSystem.String(), 0, parentID).
		Select("COALESCE(MAX(sort_no), 0)").
		Scan(&maxSortNo).Error
	if err != nil {
		return 0, fmt.Errorf("查询最大排序号失败: %w", err)
	}
	return maxSortNo, nil
}

func deleteExistingSystemRoot(ctx context.Context, db *gorm.DB, rootName string) (bool, error) {
	ctx = gormx.IgnoreTenantContext(ctx)
	var root dao.Codebook
	err := db.WithContext(ctx).
		Where("tenant_id = ? AND scope = ? AND project_id = ? AND parent_id = ? AND name = ?",
			ctxutil.SystemTenantID, domain.CodebookScopeSystem.String(), 0, 0, rootName).
		First(&root).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("查询同名 SYSTEM 根目录失败: %w", err)
	}
	if _, err = dao.NewGORMCodebookDAO(db).Delete(ctx, root.ID); err != nil {
		return false, fmt.Errorf("删除同名 SYSTEM 根目录失败: %w", err)
	}
	return true, nil
}

func shouldSkipImportEntry(entry fs.DirEntry) bool {
	switch entry.Name() {
	case ".git", ".DS_Store":
		return true
	default:
		return false
	}
}
