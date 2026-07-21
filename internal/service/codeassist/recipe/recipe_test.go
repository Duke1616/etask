package recipe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCatalogLoadsRecipes(t *testing.T) {
	catalog := NewCatalog()
	require.Len(t, catalog.items, 4)

	testCases := []struct {
		id             string
		requiresFile   bool
		allowsProposal bool
	}{
		{id: GeneralID, allowsProposal: true},
		{id: "codebook.review", requiresFile: true},
		{id: "codebook.edit", requiresFile: true, allowsProposal: true},
		{id: "codebook.legacy-migration", requiresFile: true, allowsProposal: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.id, func(t *testing.T) {
			definition, err := catalog.Get(testCase.id)
			require.NoError(t, err)
			require.Equal(t, testCase.requiresFile, definition.RequiresFileContext)
			require.Equal(t, testCase.allowsProposal, definition.AllowsCodeSuggestion)
			require.NotEmpty(t, definition.Instructions)
		})
	}
}

func TestCatalogUsesGeneralRecipeByDefault(t *testing.T) {
	definition, err := NewCatalog().Get("")
	require.NoError(t, err)
	require.Equal(t, GeneralID, definition.ID)
}
