package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/audio"
	"github.com/stashapp/stash/pkg/file"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
	"github.com/stashapp/stash/pkg/utils"
)

// used to refetch audio after hooks run
func (r *mutationResolver) getAudio(ctx context.Context, id int) (ret *models.Audio, err error) {
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.Audio.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) AudioCreate(ctx context.Context, input models.AudioCreateInput) (ret *models.Audio, err error) {
	// Process cover image if provided
	var coverImageData []byte
	if input.CoverImage != nil {
		var err error
		coverImageData, err = utils.ProcessImageInput(ctx, *input.CoverImage)
		if err != nil {
			return nil, fmt.Errorf("processing cover image: %w", err)
		}
	}

	// Create audio service instance
	audioService := &audio.Service{
		File:       r.repository.File,
		Repository: r.repository.Audio,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = audioService.CreateFromInput(ctx, input)
		if err != nil {
			return err
		}

		// Set cover image if provided
		if len(coverImageData) > 0 {
			qb := r.repository.Audio
			if err := qb.UpdateCover(ctx, ret.ID, coverImageData); err != nil {
				return fmt.Errorf("updating cover image: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside txn
	r.hookExecutor.ExecutePostHooks(ctx, ret.ID, hook.AudioCreatePost, input, nil)
	return r.getAudio(ctx, ret.ID)
}

func (r *mutationResolver) AudioUpdate(ctx context.Context, input models.AudioUpdateInput) (ret *models.Audio, err error) {
	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Start the transaction and save the audio
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		ret, err = r.audioUpdate(ctx, input, translator)
		return err
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside txn
	r.hookExecutor.ExecutePostHooks(ctx, ret.ID, hook.AudioUpdatePost, input, translator.getFields())
	return r.getAudio(ctx, ret.ID)
}

func (r *mutationResolver) AudiosUpdate(ctx context.Context, input []*models.AudioUpdateInput) (ret []*models.Audio, err error) {
	inputMaps := getUpdateInputMaps(ctx)

	// Start the transaction and save the audios
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		for i, audio := range input {
			translator := changesetTranslator{
				inputMap: inputMaps[i],
			}

			thisAudio, err := r.audioUpdate(ctx, *audio, translator)
			if err != nil {
				return err
			}

			ret = append(ret, thisAudio)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside txn
	var newRet []*models.Audio
	for i, audio := range ret {
		translator := changesetTranslator{
			inputMap: inputMaps[i],
		}

		r.hookExecutor.ExecutePostHooks(ctx, audio.ID, hook.AudioUpdatePost, input, translator.getFields())

		audio, err = r.getAudio(ctx, audio.ID)
		if err != nil {
			return nil, err
		}

		newRet = append(newRet, audio)
	}

	return newRet, nil
}

func audioPartialFromInput(input models.AudioUpdateInput, translator changesetTranslator) (*models.AudioPartial, error) {
	updatedAudio := models.NewAudioPartial()

	updatedAudio.Title = translator.optionalString(input.Title, "title")
	updatedAudio.Details = translator.optionalString(input.Details, "details")
	updatedAudio.URLs = translator.optionalURLs(input.URLs, nil)
	updatedAudio.Rating = translator.optionalInt(input.Rating100, "rating100")
	updatedAudio.Organized = translator.optionalBool(input.Organized, "organized")
	updatedAudio.ResumeTime = translator.optionalFloat64(input.ResumeTime, "resume_time")
	updatedAudio.PlayDuration = translator.optionalFloat64(input.PlayDuration, "play_duration")

	var err error
	updatedAudio.Date, err = translator.optionalDate(input.Date, "date")
	if err != nil {
		return nil, fmt.Errorf("converting date: %w", err)
	}

	updatedAudio.PrimaryFileID, err = translator.fileIDPtrFromString(input.PrimaryFileID)
	if err != nil {
		return nil, fmt.Errorf("converting primary file id: %w", err)
	}

	updatedAudio.PerformerIDs, err = translator.updateIds(input.PerformerIds, "performer_ids")
	if err != nil {
		return nil, fmt.Errorf("converting performer ids: %w", err)
	}
	updatedAudio.TagIDs, err = translator.updateIds(input.TagIds, "tag_ids")
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	return &updatedAudio, nil
}

func (r *mutationResolver) audioUpdate(ctx context.Context, input models.AudioUpdateInput, translator changesetTranslator) (*models.Audio, error) {
	audioID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	a, err := r.repository.Audio.Find(ctx, audioID)
	if err != nil {
		return nil, err
	}

	if a == nil {
		return nil, fmt.Errorf("audio with id %d not found", audioID)
	}

	// Process cover image if provided
	var coverImageData []byte
	coverIncluded := translator.hasField("cover_image")
	if input.CoverImage != nil {
		var err error
		coverImageData, err = utils.ProcessImageInput(ctx, *input.CoverImage)
		if err != nil {
			return nil, fmt.Errorf("processing cover image: %w", err)
		}
	}

	// Populate audio from the input
	updatedAudio, err := audioPartialFromInput(input, translator)
	if err != nil {
		return nil, err
	}

	if updatedAudio.PrimaryFileID != nil {
		primaryFileID := *updatedAudio.PrimaryFileID

		if err := a.LoadFiles(ctx, r.repository.Audio); err != nil {
			return nil, err
		}

		// ensure that new primary file is associated with audio
		var f models.File
		for _, ff := range a.Files.List() {
			if ff.Base().ID == primaryFileID {
				f = ff
			}
		}

		if f == nil {
			return nil, fmt.Errorf("file with id %d not associated with audio", primaryFileID)
		}
	}

	qb := r.repository.Audio
	audio, err := qb.UpdatePartial(ctx, audioID, *updatedAudio)
	if err != nil {
		return nil, err
	}

	// Update cover image if provided (including removal when set to null)
	if coverIncluded {
		if err := qb.UpdateCover(ctx, audioID, coverImageData); err != nil {
			return nil, fmt.Errorf("updating cover image: %w", err)
		}
	}

	return audio, nil
}

func (r *mutationResolver) BulkAudioUpdate(ctx context.Context, input BulkAudioUpdateInput) (ret []*models.Audio, err error) {
	audioIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
	if err != nil {
		return nil, fmt.Errorf("converting ids: %w", err)
	}

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Start the transaction and save the audios
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		for _, audioID := range audioIDs {
			updatedAudio := models.NewAudioPartial()

			updatedAudio.Title = translator.optionalString(input.Title, "title")
			updatedAudio.Details = translator.optionalString(input.Details, "details")
			updatedAudio.URLs = translator.optionalURLsBulk(input.Urls, nil)
			updatedAudio.Rating = translator.optionalInt(input.Rating100, "rating100")
			updatedAudio.Organized = translator.optionalBool(input.Organized, "organized")
			updatedAudio.ResumeTime = translator.optionalFloat64(input.ResumeTime, "resume_time")
			updatedAudio.PlayDuration = translator.optionalFloat64(input.PlayDuration, "play_duration")

			updatedAudio.Date, err = translator.optionalDate(input.Date, "date")
			if err != nil {
				return fmt.Errorf("converting date: %w", err)
			}

			updatedAudio.PerformerIDs, err = translator.updateIdsBulk(input.PerformerIds, "performer_ids")
			if err != nil {
				return fmt.Errorf("converting performer ids: %w", err)
			}
			updatedAudio.TagIDs, err = translator.updateIdsBulk(input.TagIds, "tag_ids")
			if err != nil {
				return fmt.Errorf("converting tag ids: %w", err)
			}

			a, err := r.repository.Audio.Find(ctx, audioID)
			if err != nil {
				return err
			}

			if a == nil {
				return fmt.Errorf("audio with id %d not found", audioID)
			}

			audio, err := qb.UpdatePartial(ctx, audioID, updatedAudio)
			if err != nil {
				return err
			}

			ret = append(ret, audio)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// execute post hooks outside of txn
	var newRet []*models.Audio
	for _, audio := range ret {
		r.hookExecutor.ExecutePostHooks(ctx, audio.ID, hook.AudioUpdatePost, input, translator.getFields())

		audio, err = r.getAudio(ctx, audio.ID)
		if err != nil {
			return nil, err
		}

		newRet = append(newRet, audio)
	}

	return newRet, nil
}

func (r *mutationResolver) AudioDestroy(ctx context.Context, input models.AudioDestroyInput) (ret bool, err error) {
	audioID, err := strconv.Atoi(input.ID)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	var a *models.Audio
	fileDeleter := &audio.FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   manager.GetInstance().Paths,
	}

	// Create audio service instance
	audioService := &audio.Service{
		File:       r.repository.File,
		Repository: r.repository.Audio,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		a, err = r.repository.Audio.Find(ctx, audioID)
		if err != nil {
			return err
		}

		if a == nil {
			return fmt.Errorf("audio with id %d not found", audioID)
		}

		return audioService.Destroy(ctx, a, fileDeleter, utils.IsTrue(input.DeleteGenerated), utils.IsTrue(input.DeleteFile))
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	// call post hook after performing the other actions
	r.hookExecutor.ExecutePostHooks(ctx, a.ID, hook.AudioDestroyPost, plugin.AudioDestroyInput{
		AudioDestroyInput: input,
		Checksum:          a.Checksum,
		Path:              a.Path,
	}, nil)

	return true, nil
}

func (r *mutationResolver) AudiosDestroy(ctx context.Context, input models.AudiosDestroyInput) (ret bool, err error) {
	var audios []*models.Audio
	fileDeleter := &audio.FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   manager.GetInstance().Paths,
	}

	// Create audio service instance
	audioService := &audio.Service{
		File:       r.repository.File,
		Repository: r.repository.Audio,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		err = audioService.DestroyFromInput(ctx, input, fileDeleter)
		if err != nil {
			return err
		}

		// Get audios for hooks
		audioIDs, err := stringslice.StringSliceToIntSlice(input.Ids)
		if err != nil {
			return fmt.Errorf("converting ids: %w", err)
		}

		for _, audioID := range audioIDs {
			a, err := r.repository.Audio.Find(ctx, audioID)
			if err != nil {
				// Audio might already be deleted, continue
				continue
			}
			if a != nil {
				audios = append(audios, a)
			}
		}

		return nil
	}); err != nil {
		fileDeleter.Rollback()
		return false, err
	}

	// perform the post-commit actions
	fileDeleter.Commit()

	for _, audio := range audios {
		// call post hook after performing the other actions
		r.hookExecutor.ExecutePostHooks(ctx, audio.ID, hook.AudioDestroyPost, plugin.AudiosDestroyInput{
			AudiosDestroyInput: input,
			Checksum:           audio.Checksum,
			Path:               audio.Path,
		}, nil)
	}

	return true, nil
}

func (r *mutationResolver) AudioIncrementO(ctx context.Context, id string) (ret int, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		updatedTimes, err = qb.AddO(ctx, audioID, nil)
		return err
	}); err != nil {
		return 0, err
	}

	return len(updatedTimes), nil
}

func (r *mutationResolver) AudioDecrementO(ctx context.Context, id string) (ret int, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		updatedTimes, err = qb.DeleteO(ctx, audioID, nil)
		return err
	}); err != nil {
		return 0, err
	}

	return len(updatedTimes), nil
}

func (r *mutationResolver) AudioResetO(ctx context.Context, id string) (ret int, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		// Use the oDateManager's ResetO method
		ret, err = qb.ResetO(ctx, audioID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) AudioSaveActivity(ctx context.Context, id string, resumeTime *float64, playDuration *float64) (ret bool, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		ret, err = qb.SaveActivity(ctx, audioID, resumeTime, playDuration)
		return err
	}); err != nil {
		return false, err
	}

	return ret, nil
}

func (r *mutationResolver) AudioResetActivity(ctx context.Context, id string, resetResume *bool, resetDuration *bool) (ret bool, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return false, fmt.Errorf("converting id: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		ret, err = qb.ResetActivity(ctx, audioID, utils.IsTrue(resetResume), utils.IsTrue(resetDuration))
		return err
	}); err != nil {
		return false, err
	}

	return ret, nil
}

func (r *mutationResolver) AudioAddPlay(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time

	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		updatedTimes, err = qb.AddViews(ctx, audioID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) AudioDeletePlay(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var times []time.Time

	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		updatedTimes, err = qb.DeleteViews(ctx, audioID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) AudioIncrementPlayCount(ctx context.Context, id string) (ret int, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("converting id: %w", err)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		updatedTimes, err = qb.AddViews(ctx, audioID, nil)
		return err
	}); err != nil {
		return 0, err
	}

	return len(updatedTimes), nil
}

func (r *mutationResolver) AudioResetPlayCount(ctx context.Context, id string) (ret int, err error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		ret, err = qb.DeleteAllViews(ctx, audioID)
		return err
	}); err != nil {
		return 0, err
	}

	return ret, nil
}

func (r *mutationResolver) AudioAddO(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time
	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		// Check if audio exists
		audio, err := qb.Find(ctx, audioID)
		if err != nil {
			return err
		}
		if audio == nil {
			return fmt.Errorf("audio with id %d not found", audioID)
		}

		updatedTimes, err = qb.AddO(ctx, audioID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) AudioDeleteO(ctx context.Context, id string, t []*time.Time) (*HistoryMutationResult, error) {
	audioID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	var times []time.Time
	for _, tt := range t {
		times = append(times, *tt)
	}

	var updatedTimes []time.Time

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.Audio

		// Check if audio exists
		audio, err := qb.Find(ctx, audioID)
		if err != nil {
			return err
		}
		if audio == nil {
			return fmt.Errorf("audio with id %d not found", audioID)
		}

		updatedTimes, err = qb.DeleteO(ctx, audioID, times)
		return err
	}); err != nil {
		return nil, err
	}

	return &HistoryMutationResult{
		Count:   len(updatedTimes),
		History: sliceutil.ValuesToPtrs(updatedTimes),
	}, nil
}

func (r *mutationResolver) AudioMerge(ctx context.Context, input AudioMergeInput) (*models.Audio, error) {
	srcIDs, err := stringslice.StringSliceToIntSlice(input.Source)
	if err != nil {
		return nil, fmt.Errorf("converting source ids: %w", err)
	}

	destID, err := strconv.Atoi(input.Destination)
	if err != nil {
		return nil, fmt.Errorf("converting destination id: %w", err)
	}

	var values *models.AudioPartial
	var coverImageData []byte

	if input.Values != nil {
		translator := changesetTranslator{
			inputMap: getNamedUpdateInputMap(ctx, "input.values"),
		}

		values, err = audioPartialFromInput(*input.Values, translator)
		if err != nil {
			return nil, err
		}

		if input.Values.CoverImage != nil {
			var err error
			coverImageData, err = utils.ProcessImageInput(ctx, *input.Values.CoverImage)
			if err != nil {
				return nil, fmt.Errorf("processing cover image: %w", err)
			}
		}
	} else {
		v := models.NewAudioPartial()
		values = &v
	}

	mgr := manager.GetInstance()
	fileDeleter := &audio.FileDeleter{
		Deleter: file.NewDeleter(),
		Paths:   mgr.Paths,
	}

	// Create audio service instance
	audioService := &audio.Service{
		File:       r.repository.File,
		Repository: r.repository.Audio,
	}

	var ret *models.Audio
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		if err := audioService.Merge(ctx, srcIDs, destID, fileDeleter, audio.MergeOptions{
			AudioPartial:       *values,
			IncludePlayHistory: utils.IsTrue(input.PlayHistory),
			IncludeOHistory:    utils.IsTrue(input.OHistory),
		}); err != nil {
			return err
		}

		ret, err = r.repository.Audio.Find(ctx, destID)
		if err != nil {
			return err
		}
		if ret == nil {
			return fmt.Errorf("audio with id %d not found", destID)
		}

		// update cover image if provided
		if len(coverImageData) > 0 {
			if err := r.repository.Audio.UpdateCover(ctx, destID, coverImageData); err != nil {
				return fmt.Errorf("updating cover image: %w", err)
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *mutationResolver) AudioAssignFile(ctx context.Context, input AssignAudioFileInput) (bool, error) {
	audioID, err := strconv.Atoi(input.AudioID)
	if err != nil {
		return false, fmt.Errorf("converting audio id: %w", err)
	}

	fileID, err := strconv.Atoi(input.FileID)
	if err != nil {
		return false, fmt.Errorf("converting file id: %w", err)
	}

	// Create audio service instance
	audioService := &audio.Service{
		File:       r.repository.File,
		Repository: r.repository.Audio,
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		return audioService.AssignFile(ctx, audioID, models.FileID(fileID))
	}); err != nil {
		return false, fmt.Errorf("assigning file to audio: %w", err)
	}

	return true, nil
}
