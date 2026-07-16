package codebook

import (
	"context"
	"fmt"
	"strings"

	"github.com/Duke1616/etask/internal/domain"
)

// SystemImporter 将准备好的本地代码树同步到 SYSTEM 代码资源空间。
type SystemImporter struct {
	svc Service
}

// NewSystemImporter 创建 SYSTEM 代码资源导入服务。
func NewSystemImporter(svc Service) *SystemImporter {
	return &SystemImporter{svc: svc}
}

// Import 在调用方提供的事务中应用 SYSTEM 导入计划。
func (i *SystemImporter) Import(ctx context.Context, plan SystemImportPlan) (SystemImportResult, error) {
	if plan.root.Kind != domain.CodebookKindDirectory || plan.root.Name == "" {
		return SystemImportResult{}, fmt.Errorf("SYSTEM 导入计划非法")
	}
	syncer := systemImportSyncer{
		svc: i.svc,
		result: SystemImportResult{
			RootName:  plan.request.RootName,
			SourceDir: plan.request.SourceDir,
			Skipped:   plan.skipped,
		},
	}

	// 根节点按名称匹配；replace 只删除同名子树，不影响其他 SYSTEM 组件库。
	roots, err := syncer.loadChildren(ctx, 0)
	if err != nil {
		return syncer.result, err
	}
	existing, exists := roots[plan.root.Name]
	if plan.request.Replace && exists {
		if _, err = i.svc.Delete(ctx, existing.ID); err != nil {
			return syncer.result, fmt.Errorf("删除同名 SYSTEM 根目录失败: %w", err)
		}
		syncer.result.Replaced = true
		existing, exists = domain.Codebook{}, false
	}

	root, err := syncer.syncDirectory(ctx, 0, plan.root, existing, exists)
	if err != nil {
		return syncer.result, err
	}
	syncer.result.RootID = root.ID
	return syncer.result, nil
}

type systemImportSyncer struct {
	svc    Service
	result SystemImportResult
}

func (s *systemImportSyncer) syncDirectory(ctx context.Context, parentID int64, source systemImportNode,
	existing domain.Codebook, exists bool) (domain.Codebook, error) {
	directory, created, err := s.ensureDirectory(ctx, parentID, source, existing, exists)
	if err != nil {
		return domain.Codebook{}, err
	}
	s.result.Directories++
	s.recordNode(created)

	// 已有目录一次性加载子节点，递归同步过程中不做逐文件名称查询。
	children := make(map[string]domain.Codebook)
	if !created {
		children, err = s.loadChildren(ctx, directory.ID)
		if err != nil {
			return domain.Codebook{}, err
		}
	}
	for _, child := range source.Children {
		old, found := children[child.Name]
		if child.Kind == domain.CodebookKindDirectory {
			if _, err = s.syncDirectory(ctx, directory.ID, child, old, found); err != nil {
				return domain.Codebook{}, err
			}
			continue
		}
		if err = s.syncFile(ctx, directory.ID, child, old, found); err != nil {
			return domain.Codebook{}, err
		}
	}
	return directory, nil
}

func (s *systemImportSyncer) ensureDirectory(ctx context.Context, parentID int64, source systemImportNode,
	existing domain.Codebook, exists bool) (domain.Codebook, bool, error) {
	if exists {
		if !existing.IsDirectory() {
			return domain.Codebook{}, false, fmt.Errorf("同名节点已存在但不是目录: %s", source.Path)
		}
		return existing, false, nil
	}
	directory := domain.Codebook{
		Scope: domain.CodebookScopeSystem, ProjectID: 0, ParentID: parentID,
		Name: source.Name, Kind: domain.CodebookKindDirectory,
	}
	id, err := s.svc.Create(ctx, directory)
	if err != nil {
		return domain.Codebook{}, false, fmt.Errorf("创建目录失败 %s: %w", source.Path, err)
	}
	directory.ID = id
	return directory, true, nil
}

func (s *systemImportSyncer) syncFile(ctx context.Context, parentID int64, source systemImportNode,
	existing domain.Codebook, exists bool) error {
	s.result.Files++
	if !exists {
		file := domain.Codebook{
			Scope: domain.CodebookScopeSystem, ProjectID: 0, ParentID: parentID,
			Name: source.Name, Kind: domain.CodebookKindFile, Code: source.Code,
		}
		if _, err := s.svc.Create(ctx, file); err != nil {
			return fmt.Errorf("创建文件失败 %s: %w", source.Path, err)
		}
		s.result.Created++
		return nil
	}
	if !existing.IsFile() {
		return fmt.Errorf("同名节点已存在但不是文件: %s", source.Path)
	}
	current, err := s.svc.GetByID(ctx, existing.ID)
	if err != nil {
		return fmt.Errorf("读取已有文件失败 %s: %w", source.Path, err)
	}
	if current.Code == source.Code {
		s.result.Unchanged++
		return nil
	}
	// 内容变化创建新版本并切换 current_version，保留已有版本历史。
	if strings.TrimSpace(source.Code) == "" {
		return fmt.Errorf("文件内容为空，无法为已有文件创建新版本: %s", source.Path)
	}
	versionID, err := s.svc.CreateVersion(ctx, domain.CodebookVersion{
		NodeID: existing.ID, Code: source.Code, Message: "SYSTEM 组件库增量导入",
	})
	if err != nil {
		return fmt.Errorf("创建文件新版本失败 %s: %w", source.Path, err)
	}
	if _, err = s.svc.UseVersion(ctx, existing.ID, versionID); err != nil {
		return fmt.Errorf("切换文件当前版本失败 %s: %w", source.Path, err)
	}
	s.result.Updated++
	return nil
}

func (s *systemImportSyncer) loadChildren(ctx context.Context, parentID int64) (map[string]domain.Codebook, error) {
	nodes, err := s.svc.Children(ctx, 0, parentID)
	if err != nil {
		return nil, fmt.Errorf("查询目录子节点失败: %w", err)
	}
	children := make(map[string]domain.Codebook, len(nodes))
	for _, node := range nodes {
		children[node.Name] = node
	}
	return children, nil
}

func (s *systemImportSyncer) recordNode(created bool) {
	if created {
		s.result.Created++
		return
	}
	s.result.Unchanged++
}
