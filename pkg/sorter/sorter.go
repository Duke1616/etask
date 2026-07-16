package sorter

import (
	"slices"

	"github.com/ecodeclub/ekit/slice"
)

const (
	// DefaultIndexGap 默认稀疏排序间隔。
	DefaultIndexGap int64 = 1000
)

// Sortable 定义可拖拽排序元素。
type Sortable interface {
	// GetID 返回排序元素 ID。
	GetID() int64
	// GetSortKey 返回当前排序键。
	GetSortKey() int64
}

// SortItem 定义排序更新项。
type SortItem interface {
	any
}

// ReorderPlan 表示一次拖拽重排的执行计划。
type ReorderPlan[T SortItem] struct {
	NeedRebalance bool
	NewSortKey    int64
	Items         []T
}

// Sorter 基于稀疏索引计算拖拽排序结果。
type Sorter[E Sortable, T SortItem] struct {
	indexGap    int64
	convertFunc func(elem E, idx int) T
}

// NewSorter 创建排序器。
func NewSorter[E Sortable, T SortItem](convertFunc func(elem E, idx int) T) *Sorter[E, T] {
	return &Sorter[E, T]{
		indexGap:    DefaultIndexGap,
		convertFunc: convertFunc,
	}
}

// WithIndexGap 设置排序间隔。
func (s *Sorter[E, T]) WithIndexGap(gap int64) *Sorter[E, T] {
	s.indexGap = gap
	return s
}

// PlanReorder 计算拖拽元素移动到目标位置后的排序计划，targetPosition 从 0 开始。
func (s *Sorter[E, T]) PlanReorder(elements []E, draggedElem E, targetPosition int64) ReorderPlan[T] {
	remainingElems := s.removeDragged(elements, draggedElem.GetID())
	newSortKey := s.calculateSortKey(remainingElems, targetPosition)
	if s.needsRebalance(remainingElems, targetPosition, newSortKey) {
		finalList := s.insertElem(remainingElems, draggedElem, targetPosition)
		return ReorderPlan[T]{
			NeedRebalance: true,
			Items:         s.generateRebalanceItems(finalList),
		}
	}
	return ReorderPlan[T]{
		NewSortKey: newSortKey,
	}
}

func (s *Sorter[E, T]) removeDragged(elems []E, draggedID int64) []E {
	idx := slices.IndexFunc(elems, func(e E) bool {
		return e.GetID() == draggedID
	})
	if idx == -1 {
		return elems
	}
	return slices.Delete(slices.Clone(elems), idx, idx+1)
}

func (s *Sorter[E, T]) insertElem(elems []E, elem E, position int64) []E {
	if position < 0 {
		position = 0
	}
	if position > int64(len(elems)) {
		position = int64(len(elems))
	}
	return slices.Insert(slices.Clone(elems), int(position), elem)
}

func (s *Sorter[E, T]) calculateSortKey(elems []E, position int64) int64 {
	n := int64(len(elems))
	if n == 0 {
		return s.indexGap
	}
	if position >= n {
		return elems[n-1].GetSortKey() + s.indexGap
	}
	if position <= 0 {
		return elems[0].GetSortKey() / 2
	}
	return (elems[position-1].GetSortKey() + elems[position].GetSortKey()) / 2
}

func (s *Sorter[E, T]) needsRebalance(elems []E, position, newSortKey int64) bool {
	if position <= 0 || position >= int64(len(elems)) {
		return false
	}
	return newSortKey <= elems[position-1].GetSortKey()
}

func (s *Sorter[E, T]) generateRebalanceItems(elems []E) []T {
	return slice.Map(elems, func(idx int, src E) T {
		return s.convertFunc(src, idx)
	})
}
