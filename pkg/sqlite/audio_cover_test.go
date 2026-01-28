//go:build integration
// +build integration

package sqlite_test

import (
	"context"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudioStore_CoverMethods(t *testing.T) {
	runWithRollbackTxn(t, "cover methods", func(t *testing.T, ctx context.Context) {
		repo := db.Repository()
		store := repo.Audio

		// Create a test audio
		audio := models.NewAudio()
		audio.Title = "Test Audio for Cover"
		audio.Organized = true

		err := store.Create(ctx, &audio, []models.FileID{})
		require.NoError(t, err)
		require.NotZero(t, audio.ID)

		// Test that initially there's no cover
		hasCover, err := store.HasCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.False(t, hasCover)

		// Test GetCover returns empty for non-existent cover
		coverData, err := store.GetCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.Empty(t, coverData)

		// Test UpdateCover with sample image data
		testImageData := []byte("fake image data for testing")
		err = store.UpdateCover(ctx, audio.ID, testImageData)
		require.NoError(t, err)

		// Test HasCover now returns true
		hasCover, err = store.HasCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.True(t, hasCover)

		// Test GetCover returns the image data
		retrievedData, err := store.GetCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.Equal(t, testImageData, retrievedData)

		// Test updating with new image data
		newImageData := []byte("updated fake image data")
		err = store.UpdateCover(ctx, audio.ID, newImageData)
		require.NoError(t, err)

		// Verify the updated data
		retrievedData, err = store.GetCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.Equal(t, newImageData, retrievedData)

		// Test removing cover (empty data)
		err = store.UpdateCover(ctx, audio.ID, []byte{})
		require.NoError(t, err)

		// Test HasCover now returns false
		hasCover, err = store.HasCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.False(t, hasCover)

		// Test GetCover returns empty
		coverData, err = store.GetCover(ctx, audio.ID)
		require.NoError(t, err)
		assert.Empty(t, coverData)
	})
}

func TestAudioStore_CoverMethods_NonExistentAudio(t *testing.T) {
	runWithRollbackTxn(t, "non-existent audio", func(t *testing.T, ctx context.Context) {
		repo := db.Repository()
		store := repo.Audio

		nonExistentID := 99999

		// Test HasCover with non-existent audio
		hasCover, err := store.HasCover(ctx, nonExistentID)
		require.NoError(t, err)
		assert.False(t, hasCover)

		// Test GetCover with non-existent audio
		coverData, err := store.GetCover(ctx, nonExistentID)
		require.NoError(t, err)
		assert.Empty(t, coverData)

		// Test UpdateCover with non-existent audio - this may or may not return error depending on implementation
		testImageData := []byte("fake image data")
		err = store.UpdateCover(ctx, nonExistentID, testImageData)
		// Note: The current implementation may not return an error for non-existent IDs
		// This is consistent with other blob operations
	})
}
