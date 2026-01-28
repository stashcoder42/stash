package audio

import (
	"context"
	"strconv"

	"github.com/stashapp/stash/pkg/models"
)

type QueryCounter interface {
	QueryCount(ctx context.Context, audioFilter *models.AudioFilterType, findFilter *models.FindFilterType) (int, error)
}

func CountByTagID(ctx context.Context, r QueryCounter, id int, depth *int) (int, error) {
	filter := &models.AudioFilterType{
		Tags: &models.HierarchicalMultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
			Depth:    depth,
		},
	}

	return r.QueryCount(ctx, filter, nil)
}

func CountByPerformerID(ctx context.Context, r QueryCounter, id int) (int, error) {
	filter := &models.AudioFilterType{
		Performers: &models.MultiCriterionInput{
			Value:    []string{strconv.Itoa(id)},
			Modifier: models.CriterionModifierIncludes,
		},
	}

	return r.QueryCount(ctx, filter, nil)
}
