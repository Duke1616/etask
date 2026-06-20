package sorter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testItem struct {
	id      int64
	sortKey int64
}

func (t testItem) GetID() int64 {
	return t.id
}

func (t testItem) GetSortKey() int64 {
	return t.sortKey
}

func TestSorterPlanReorderFastPath(t *testing.T) {
	s := NewSorter[testItem, testItem](func(elem testItem, idx int) testItem {
		elem.sortKey = int64(idx+1) * DefaultIndexGap
		return elem
	})
	plan := s.PlanReorder([]testItem{
		{id: 1, sortKey: 1000},
		{id: 2, sortKey: 2000},
		{id: 3, sortKey: 3000},
	}, testItem{id: 3, sortKey: 3000}, 1)

	require.False(t, plan.NeedRebalance)
	require.Equal(t, int64(1500), plan.NewSortKey)
}

func TestSorterPlanReorderRebalance(t *testing.T) {
	s := NewSorter[testItem, testItem](func(elem testItem, idx int) testItem {
		elem.sortKey = int64(idx+1) * DefaultIndexGap
		return elem
	})
	plan := s.PlanReorder([]testItem{
		{id: 1, sortKey: 1},
		{id: 2, sortKey: 2},
		{id: 3, sortKey: 3},
	}, testItem{id: 3, sortKey: 3}, 1)

	require.True(t, plan.NeedRebalance)
	require.Equal(t, []testItem{
		{id: 1, sortKey: 1000},
		{id: 3, sortKey: 2000},
		{id: 2, sortKey: 3000},
	}, plan.Items)
}

func TestSorterPlanReorderClampPosition(t *testing.T) {
	s := NewSorter[testItem, testItem](func(elem testItem, idx int) testItem {
		elem.sortKey = int64(idx+1) * DefaultIndexGap
		return elem
	})
	plan := s.PlanReorder([]testItem{
		{id: 1, sortKey: 1000},
		{id: 2, sortKey: 2000},
	}, testItem{id: 3}, -1)

	require.False(t, plan.NeedRebalance)
	require.Equal(t, int64(500), plan.NewSortKey)
}
