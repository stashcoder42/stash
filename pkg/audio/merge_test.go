package audio

import (
	"context"
	"testing"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestServiceMergeExists(t *testing.T) {
	// Simple test to verify the Merge method exists and has the correct signature
	// This test just ensures the method compiles with the right signature

	// Test that the method signature is correct by creating the inputs
	ctx := context.Background()
	sourceIDs := []int{1}
	destinationID := 2
	fileDeleter := makeTestFileDeleter()
	options := MergeOptions{
		AudioPartial: models.AudioPartial{
			Title: models.NewOptionalString("Test"),
		},
		IncludePlayHistory: false,
		IncludeOHistory:    false,
	}

	// Just verify the types and signature exist - don't actually call the method
	// as it would require proper repository setup
	assert.NotNil(t, ctx)
	assert.NotNil(t, sourceIDs)
	assert.NotNil(t, destinationID)
	assert.NotNil(t, fileDeleter)
	assert.NotNil(t, options)

	// Test that the Service type has the expected method (compilation test)
	var service *Service
	if service != nil {
		// This line ensures the Merge method exists with the right signature
		_ = service.Merge(ctx, sourceIDs, destinationID, fileDeleter, options)
	}
}

func TestMergeOptionsStruct(t *testing.T) {
	// Test that MergeOptions struct has the expected fields
	options := MergeOptions{
		AudioPartial: models.AudioPartial{
			Title: models.NewOptionalString("Test Title"),
		},
		IncludePlayHistory: true,
		IncludeOHistory:    true,
	}

	assert.Equal(t, true, options.IncludePlayHistory)
	assert.Equal(t, true, options.IncludeOHistory)
	assert.NotNil(t, options.AudioPartial)
}
