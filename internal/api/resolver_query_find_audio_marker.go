package api

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/sliceutil/stringslice"
)

func (r *queryResolver) FindAudioMarkers(ctx context.Context, audioMarkerFilter *models.AudioMarkerFilterType, filter *models.FindFilterType, ids []string) (ret *FindAudioMarkersResultType, err error) {
	idInts, err := stringslice.StringSliceToIntSlice(ids)
	if err != nil {
		return nil, err
	}

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		var audioMarkers []*models.AudioMarker
		var err error
		var total int

		if len(idInts) > 0 {
			audioMarkers, err = r.repository.AudioMarker.FindMany(ctx, idInts)
			total = len(audioMarkers)
		} else {
			audioMarkers, total, err = r.repository.AudioMarker.Query(ctx, audioMarkerFilter, filter)
		}

		if err != nil {
			return err
		}

		ret = &FindAudioMarkersResultType{
			Count:        total,
			AudioMarkers: audioMarkers,
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) AllAudioMarkers(ctx context.Context) (ret []*models.AudioMarker, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.AudioMarker.All(ctx)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *queryResolver) AudioMarkerStrings(ctx context.Context, q *string, sort *string) (ret []*models.MarkerStringsResultType, err error) {
	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		ret, err = r.repository.AudioMarker.GetMarkerStrings(ctx, q, sort)
		return err
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

// Get audio marker tags which show up under the audio player.
func (r *queryResolver) AudioMarkerTags(ctx context.Context, audioID string) ([]*AudioMarkerTag, error) {
	id, err := strconv.Atoi(audioID)
	if err != nil {
		return nil, err
	}

	var keys []int
	tags := make(map[int]*AudioMarkerTag)

	if err := r.withReadTxn(ctx, func(ctx context.Context) error {
		audioMarkers, err := r.repository.AudioMarker.FindByAudioID(ctx, id)
		if err != nil {
			return err
		}

		tqb := r.repository.Tag
		for _, audioMarker := range audioMarkers {
			markerPrimaryTag, err := tqb.Find(ctx, audioMarker.PrimaryTagID)
			if err != nil {
				return err
			}

			if markerPrimaryTag == nil {
				return fmt.Errorf("tag with id %d not found", audioMarker.PrimaryTagID)
			}

			_, hasKey := tags[markerPrimaryTag.ID]
			if !hasKey {
				audioMarkerTag := &AudioMarkerTag{Tag: markerPrimaryTag}
				tags[markerPrimaryTag.ID] = audioMarkerTag
				keys = append(keys, markerPrimaryTag.ID)
			}
			tags[markerPrimaryTag.ID].AudioMarkers = append(tags[markerPrimaryTag.ID].AudioMarkers, audioMarker)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// Sort so that primary tags that show up earlier in the audio are first.
	sort.Slice(keys, func(i, j int) bool {
		a := tags[keys[i]]
		b := tags[keys[j]]
		return a.AudioMarkers[0].Seconds < b.AudioMarkers[0].Seconds
	})

	var result []*AudioMarkerTag
	for _, key := range keys {
		result = append(result, tags[key])
	}

	return result, nil
}
