package api

import (
	"context"

	"github.com/stashapp/stash/pkg/models"
)

// URL resolver for AudioCreateInput (deprecated field)
func (r *audioCreateInputResolver) URL(ctx context.Context, obj *models.AudioCreateInput, data *string) error {
	// Deprecated field - this should not be used, but we provide it for backwards compatibility
	return nil
}

// URLs resolver for AudioCreateInput
func (r *audioCreateInputResolver) Urls(ctx context.Context, obj *models.AudioCreateInput, data []string) error {
	// TODO: Implement URL handling for audio create input
	return nil
}
