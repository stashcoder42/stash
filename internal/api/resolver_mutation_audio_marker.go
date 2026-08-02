package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/plugin/hook"
	"github.com/stashapp/stash/pkg/sliceutil"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

// AudioMarker mutations

func (r *mutationResolver) getAudioMarker(ctx context.Context, id int) (ret *models.AudioMarker, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.AudioMarker.Find(ctx, id)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func validateAudioMarkerEndSeconds(seconds, endSeconds float64) error {
	if endSeconds < seconds {
		return fmt.Errorf("end_seconds (%f) must be greater than or equal to seconds (%f)", endSeconds, seconds)
	}
	return nil
}

func (r *mutationResolver) AudioMarkerCreate(ctx context.Context, input AudioMarkerCreateInput) (*models.AudioMarker, error) {
	audioID, err := strconv.Atoi(input.AudioID)
	if err != nil {
		return nil, fmt.Errorf("converting audio id: %w", err)
	}

	primaryTagID, err := strconv.Atoi(input.PrimaryTagID)
	if err != nil {
		return nil, fmt.Errorf("converting primary tag id: %w", err)
	}

	// Populate a new audio marker from the input
	newMarker := models.NewAudioMarker()

	newMarker.Title = input.Title
	newMarker.Seconds = input.Seconds
	newMarker.PrimaryTagID = primaryTagID
	newMarker.AudioID = audioID

	if input.EndSeconds != nil {
		if err := validateAudioMarkerEndSeconds(newMarker.Seconds, *input.EndSeconds); err != nil {
			return nil, err
		}
		newMarker.EndSeconds = input.EndSeconds
	}

	tagIDs, err := stringslice.StringSliceToIntSlice(input.TagIds)
	if err != nil {
		return nil, fmt.Errorf("converting tag ids: %w", err)
	}

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.AudioMarker

		err := qb.Create(ctx, &newMarker)
		if err != nil {
			return err
		}

		// Save the marker tags
		// If this tag is the primary tag, then let's not add it.
		tagIDs = sliceutil.Exclude(tagIDs, []int{newMarker.PrimaryTagID})
		return qb.UpdateTags(ctx, newMarker.ID, tagIDs)
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, newMarker.ID, hook.AudioMarkerCreatePost, input, nil)
	return r.getAudioMarker(ctx, newMarker.ID)
}

func (r *mutationResolver) AudioMarkerUpdate(ctx context.Context, input AudioMarkerUpdateInput) (*models.AudioMarker, error) {
	markerID, err := strconv.Atoi(input.ID)
	if err != nil {
		return nil, fmt.Errorf("converting id: %w", err)
	}

	translator := changesetTranslator{
		inputMap: getUpdateInputMap(ctx),
	}

	// Populate audio marker from the input
	updatedMarker := models.NewAudioMarkerPartial()

	updatedMarker.Title = translator.optionalString(input.Title, "title")
	updatedMarker.Seconds = translator.optionalFloat64(input.Seconds, "seconds")
	updatedMarker.EndSeconds = translator.optionalFloat64(input.EndSeconds, "end_seconds")
	updatedMarker.AudioID, err = translator.optionalIntFromString(input.AudioID, "audio_id")
	if err != nil {
		return nil, fmt.Errorf("converting audio id: %w", err)
	}
	updatedMarker.PrimaryTagID, err = translator.optionalIntFromString(input.PrimaryTagID, "primary_tag_id")
	if err != nil {
		return nil, fmt.Errorf("converting primary tag id: %w", err)
	}

	var tagIDs []int
	tagIdsIncluded := translator.hasField("tag_ids")
	if input.TagIds != nil {
		tagIDs, err = stringslice.StringSliceToIntSlice(input.TagIds)
		if err != nil {
			return nil, fmt.Errorf("converting tag ids: %w", err)
		}
	}

	// Start the transaction and save the audio marker
	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.AudioMarker

		// check to see if marker exists
		existingMarker, err := qb.Find(ctx, markerID)
		if err != nil {
			return err
		}
		if existingMarker == nil {
			return fmt.Errorf("audio marker with id %d not found", markerID)
		}

		// Validate end_seconds
		shouldValidateEndSeconds := (updatedMarker.Seconds.Set || updatedMarker.EndSeconds.Set) && !updatedMarker.EndSeconds.Null
		if shouldValidateEndSeconds {
			seconds := existingMarker.Seconds
			if updatedMarker.Seconds.Set {
				seconds = updatedMarker.Seconds.Value
			}

			endSeconds := existingMarker.EndSeconds
			if updatedMarker.EndSeconds.Set {
				endSeconds = &updatedMarker.EndSeconds.Value
			}

			if endSeconds != nil {
				if err := validateAudioMarkerEndSeconds(seconds, *endSeconds); err != nil {
					return err
				}
			}
		}

		newMarker, err := qb.UpdatePartial(ctx, markerID, updatedMarker)
		if err != nil {
			return err
		}

		if tagIdsIncluded {
			// Save the marker tags
			// If this tag is the primary tag, then let's not add it.
			tagIDs = sliceutil.Exclude(tagIDs, []int{newMarker.PrimaryTagID})
			if err := qb.UpdateTags(ctx, markerID, tagIDs); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	r.hookExecutor.ExecutePostHooks(ctx, markerID, hook.AudioMarkerUpdatePost, input, translator.getFields())
	return r.getAudioMarker(ctx, markerID)
}

func (r *mutationResolver) AudioMarkerDestroy(ctx context.Context, id string) (bool, error) {
	return r.AudioMarkersDestroy(ctx, []string{id})
}

func (r *mutationResolver) AudioMarkersDestroy(ctx context.Context, markerIDs []string) (bool, error) {
	ids, err := stringslice.StringSliceToIntSlice(markerIDs)
	if err != nil {
		return false, fmt.Errorf("converting ids: %w", err)
	}

	var markers []*models.AudioMarker

	if err := r.withTxn(ctx, func(ctx context.Context) error {
		qb := r.repository.AudioMarker

		for _, markerID := range ids {
			marker, err := qb.Find(ctx, markerID)

			if err != nil {
				return err
			}

			if marker == nil {
				return fmt.Errorf("audio marker with id %d not found", markerID)
			}

			markers = append(markers, marker)

			if err := qb.Destroy(ctx, markerID); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return false, err
	}

	for _, marker := range markers {
		r.hookExecutor.ExecutePostHooks(ctx, marker.ID, hook.AudioMarkerDestroyPost, markerIDs, nil)
	}

	return true, nil
}
