package api

import (
	"context"
	"time"

	"github.com/stashapp/stash/internal/api/loaders"
	"github.com/stashapp/stash/internal/api/urlbuilders"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/models"
)

func (r *audioResolver) getFiles(ctx context.Context, obj *models.Audio) ([]models.File, error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		return obj.LoadFiles(ctx, r.repository.Audio)
	}); err != nil {
		return nil, err
	}

	return obj.Files.List(), nil
}

func (r *audioResolver) Date(ctx context.Context, obj *models.Audio) (*string, error) {
	if obj.Date != nil {
		result := obj.Date.String()
		return &result, nil
	}
	return nil, nil
}

func (r *audioResolver) Files(ctx context.Context, obj *models.Audio) ([]*AudioFile, error) {
	files, err := r.getFiles(ctx, obj)
	if err != nil {
		return nil, err
	}

	var ret []*AudioFile
	for _, f := range files {
		// filter out non-audio files
		audioFile, ok := f.(*models.AudioFile)
		if !ok {
			continue
		}
		ret = append(ret, &AudioFile{AudioFile: audioFile})
	}

	return ret, nil
}

func (r *audioResolver) Paths(ctx context.Context, obj *models.Audio) (*AudioPathsType, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	builder := urlbuilders.NewAudioURLBuilder(baseURL, obj)

	streamURL := builder.GetStreamURL("").String()
	thumbnailURL := builder.GetThumbnailURL()
	captionBasePath := builder.GetCaptionURL()

	// Check if audio has a cover before including the cover URL
	var coverURL *string
	var hasCover bool
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		hasCover, err = r.repository.Audio.HasCover(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	if hasCover {
		coverURLStr := builder.GetCoverURL()
		coverURL = &coverURLStr
	}

	return &AudioPathsType{
		Stream:  &streamURL,
		Cover:   coverURL,
		Preview: &thumbnailURL,
		Caption: &captionBasePath,
	}, nil
}

func (r *audioResolver) Rating100(ctx context.Context, obj *models.Audio) (*int, error) {
	return obj.Rating, nil
}

func (r *audioResolver) Checksum(ctx context.Context, obj *models.Audio) (*string, error) {
	if obj.Checksum != "" {
		return &obj.Checksum, nil
	}

	// Load files if not already loaded
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		return obj.LoadPrimaryFile(ctx, r.repository.File)
	}); err != nil {
		return nil, err
	}

	// Get checksum from primary file using MD5 fingerprint (consistent with audio scanning)
	primaryFile := obj.Files.Primary()
	if primaryFile != nil {
		checksum := primaryFile.Base().Fingerprints.GetString(models.FingerprintTypeMD5)
		if checksum != "" {
			return &checksum, nil
		}
	}

	return nil, nil
}

func (r *audioResolver) Tags(ctx context.Context, obj *models.Audio) (ret []*models.Tag, err error) {
	if !obj.TagIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadTagIDs(ctx, r.repository.Audio)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).TagByID.LoadAll(obj.TagIDs.List())
	return ret, firstError(errs)
}

func (r *audioResolver) Performers(ctx context.Context, obj *models.Audio) (ret []*models.Performer, err error) {
	if !obj.PerformerIDs.Loaded() {
		if err := r.withReadTxn(ctx, func(ctx context.Context) error {
			return obj.LoadPerformerIDs(ctx, r.repository.Audio)
		}); err != nil {
			return nil, err
		}
	}

	var errs []error
	ret, errs = loaders.From(ctx).PerformerByID.LoadAll(obj.PerformerIDs.List())
	return ret, firstError(errs)
}

func (r *audioResolver) URL(ctx context.Context, obj *models.Audio) (*string, error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		return obj.LoadURLs(ctx, r.repository.Audio)
	}); err != nil {
		return nil, err
	}

	urls := obj.URLs.List()
	if len(urls) == 0 {
		return nil, nil
	}
	return &urls[0], nil
}

func (r *audioResolver) Urls(ctx context.Context, obj *models.Audio) ([]string, error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		return obj.LoadURLs(ctx, r.repository.Audio)
	}); err != nil {
		return nil, err
	}

	return obj.URLs.List(), nil
}

func (r *audioResolver) PlayCount(ctx context.Context, obj *models.Audio) (*int, error) {
	ret, err := loaders.From(ctx).AudioPlayCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *audioResolver) OCounter(ctx context.Context, obj *models.Audio) (*int, error) {
	ret, err := loaders.From(ctx).AudioOCount.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

func (r *audioResolver) LastPlayedAt(ctx context.Context, obj *models.Audio) (*time.Time, error) {
	ret, err := loaders.From(ctx).AudioLastPlayed.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *audioResolver) PlayHistory(ctx context.Context, obj *models.Audio) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).AudioPlayHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i := range ret {
		ptrRet[i] = &ret[i]
	}

	return ptrRet, nil
}

func (r *audioResolver) OHistory(ctx context.Context, obj *models.Audio) ([]*time.Time, error) {
	ret, err := loaders.From(ctx).AudioOHistory.Load(obj.ID)
	if err != nil {
		return nil, err
	}

	// convert to pointer slice
	ptrRet := make([]*time.Time, len(ret))
	for i := range ret {
		ptrRet[i] = &ret[i]
	}

	return ptrRet, nil
}

func (r *audioResolver) AudioStreams(ctx context.Context, obj *models.Audio) ([]*AudioStreamEndpoint, error) {
	baseURL, _ := ctx.Value(BaseURLCtxKey).(string)
	builder := urlbuilders.NewAudioURLBuilder(baseURL, obj)

	streamURL := builder.GetStreamURL("")

	var managerEndpoints []*manager.AudioStreamEndpoint

	// Use the enhanced streaming function from manager within a transaction
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var err error
		managerEndpoints, err = manager.GetAudioStreamPaths(ctx, obj, streamURL, r.repository.File)
		return err
	}); err != nil {
		return nil, err
	}

	// Convert manager endpoints to API endpoints
	apiEndpoints := make([]*AudioStreamEndpoint, len(managerEndpoints))
	for i, ep := range managerEndpoints {
		apiEndpoints[i] = &AudioStreamEndpoint{
			URL:      ep.URL,
			MimeType: ep.MimeType,
			Label:    ep.Label,
		}
	}

	return apiEndpoints, nil
}

func (r *audioResolver) AudioMarkers(ctx context.Context, obj *models.Audio) (ret []*models.AudioMarker, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.AudioMarker.FindByAudioID(ctx, obj.ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *audioResolver) Captions(ctx context.Context, obj *models.Audio) (ret []*models.VideoCaption, err error) {
	// Load primary file if not already loaded
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		return obj.LoadPrimaryFile(ctx, r.repository.File)
	}); err != nil {
		return nil, err
	}

	primaryFile := obj.Files.Primary()
	if primaryFile == nil {
		return nil, nil
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.File.GetAudioCaptions(ctx, primaryFile.Base().ID)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, err
}
