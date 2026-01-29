package api

import (
	"context"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/pkg/models"
)

func (r *audioMarkerResolver) Audio(ctx context.Context, obj *models.AudioMarker) (ret *models.Audio, err error) {
	return loaders.From(ctx).AudioByID.Load(obj.AudioID)
}

func (r *audioMarkerResolver) PrimaryTag(ctx context.Context, obj *models.AudioMarker) (ret *models.Tag, err error) {
	return loaders.From(ctx).TagByID.Load(obj.PrimaryTagID)
}

func (r *audioMarkerResolver) Tags(ctx context.Context, obj *models.AudioMarker) (ret []*models.Tag, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Tag.FindByAudioMarkerID(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *audioMarkerResolver) Stream(ctx context.Context, obj *models.AudioMarker) (string, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	builder := urlbuilders.NewAudioMarkerURLBuilder(baseURL, obj)
	return builder.GetStreamURL(), nil
}
