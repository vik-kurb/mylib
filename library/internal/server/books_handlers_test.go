package server

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMergeAuthors_NoDiff(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := oldAuthors
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.Equal(t, len(diff.authorsToDelete), 0)
	assert.Equal(t, len(diff.authorsToInsert), 0)
}

func TestMergeAuthors_OnlyDelete(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := []uuid.UUID{oldAuthors[0]}
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.ElementsMatch(t, diff.authorsToDelete, []uuid.UUID{oldAuthors[1], oldAuthors[2]})
	assert.Equal(t, len(diff.authorsToInsert), 0)
}

func TestMergeAuthors_OnlyInsert(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := oldAuthors
	newAuthors = append(newAuthors, uuid.New(), uuid.New())
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.ElementsMatch(t, diff.authorsToInsert, []uuid.UUID{newAuthors[3], newAuthors[4]})
	assert.Equal(t, len(diff.authorsToDelete), 0)
}

func TestMergeAuthors_DeleteAndInsert(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := []uuid.UUID{oldAuthors[0], uuid.New(), uuid.New()}
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.ElementsMatch(t, diff.authorsToInsert, []uuid.UUID{newAuthors[1], newAuthors[2]})
	assert.ElementsMatch(t, diff.authorsToDelete, []uuid.UUID{oldAuthors[1], oldAuthors[2]})
}
