package codebook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Duke1616/etask/internal/domain"
)

// SystemImportRequest 描述一次 SYSTEM 组件库目录导入。
type SystemImportRequest struct {
	SourceDir string
	RootName  string
	Replace   bool
}

// SystemImportPlan 是完成文件读取后的不可变 SYSTEM 导入计划。
// 计划在数据库事务外生成，避免目录扫描和文件读取占用事务时间。
type SystemImportPlan struct {
	request SystemImportRequest
	root    systemImportNode
	skipped int
}

// SystemImportResult 汇总 SYSTEM 组件库目录导入结果。
type SystemImportResult struct {
	// RootID 是导入后 SYSTEM 根目录的代码节点 ID。
	RootID int64
	// Directories 是本次扫描并处理的目录总数，包含 SYSTEM 根目录。
	Directories int
	// Files 是本次扫描并处理的普通文件总数。
	Files int
	// Created 是本次新建的目录和文件节点总数。
	Created int
	// Updated 是因内容变化而创建并启用新版本的文件总数。
	Updated int
	// Unchanged 是复用的已有目录和内容未变化的已有文件总数。
	Unchanged int
	// Skipped 是因忽略规则或文件类型不受支持而跳过的目录和文件总数。
	Skipped int
	// Replaced 表示导入前是否删除了同名 SYSTEM 根目录及其子树。
	Replaced bool
	// RootName 是导入到 Codebook 后的 SYSTEM 根目录名称。
	RootName string
	// SourceDir 是规范化为绝对路径后的本地来源目录。
	SourceDir string
}

type systemImportNode struct {
	Name     string
	Path     string
	Kind     domain.CodebookKind
	Code     string
	Children []systemImportNode
}

// PrepareSystemImport 校验目录参数并在事务外读取完整的本地代码树。
func PrepareSystemImport(request SystemImportRequest) (SystemImportPlan, error) {
	// 先规范化并校验路径，计划中的 SourceDir 始终是稳定绝对路径。
	prepared, err := prepareSystemImportRequest(request)
	if err != nil {
		return SystemImportPlan{}, err
	}
	// 文件读取全部发生在数据库事务外，导入阶段只消费内存中的不可变计划。
	root, skipped, err := readSystemImportDirectory(prepared.SourceDir, prepared.RootName)
	if err != nil {
		return SystemImportPlan{}, err
	}
	return SystemImportPlan{request: prepared, root: root, skipped: skipped}, nil
}

func prepareSystemImportRequest(request SystemImportRequest) (SystemImportRequest, error) {
	sourceDir, err := filepath.Abs(strings.TrimSpace(request.SourceDir))
	if err != nil {
		return SystemImportRequest{}, fmt.Errorf("解析导入目录失败: %w", err)
	}
	info, err := os.Stat(sourceDir)
	if err != nil {
		return SystemImportRequest{}, fmt.Errorf("读取导入目录失败: %w", err)
	}
	if !info.IsDir() {
		return SystemImportRequest{}, fmt.Errorf("导入路径必须是目录: %s", sourceDir)
	}

	rootName := strings.TrimSpace(request.RootName)
	if rootName == "" {
		rootName = filepath.Base(sourceDir)
	}
	if rootName == "" || rootName == "." || rootName == string(os.PathSeparator) {
		return SystemImportRequest{}, fmt.Errorf("根目录名称不能为空")
	}
	if strings.ContainsAny(rootName, `/\`) {
		return SystemImportRequest{}, fmt.Errorf("根目录名称不能包含路径分隔符: %s", rootName)
	}
	request.SourceDir = sourceDir
	request.RootName = rootName
	return request, nil
}

func readSystemImportDirectory(path, name string) (systemImportNode, int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return systemImportNode{}, 0, fmt.Errorf("读取目录失败 %s: %w", path, err)
	}
	node := systemImportNode{Name: name, Path: path, Kind: domain.CodebookKindDirectory}
	skipped := 0
	for _, entry := range entries {
		// 忽略版本控制元数据，并跳过符号链接等非普通文件。
		if skipSystemImportEntry(entry.Name()) {
			skipped++
			continue
		}
		entryPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			child, childSkipped, readErr := readSystemImportDirectory(entryPath, entry.Name())
			if readErr != nil {
				return systemImportNode{}, 0, readErr
			}
			node.Children = append(node.Children, child)
			skipped += childSkipped
			continue
		}
		if !entry.Type().IsRegular() {
			skipped++
			continue
		}
		code, readErr := os.ReadFile(entryPath)
		if readErr != nil {
			return systemImportNode{}, 0, fmt.Errorf("读取文件失败 %s: %w", entryPath, readErr)
		}
		node.Children = append(node.Children, systemImportNode{
			Name: entry.Name(), Path: entryPath, Kind: domain.CodebookKindFile, Code: string(code),
		})
	}
	return node, skipped, nil
}

func skipSystemImportEntry(name string) bool {
	return name == ".git" || name == ".DS_Store"
}
