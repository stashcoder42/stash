package sqlite

import (
	"context"
	"fmt"

	"github.com/stashapp/stash/pkg/models"
)

type audioMarkerFilterHandler struct {
	audioMarkerFilter *models.AudioMarkerFilterType
}

func (qb *audioMarkerFilterHandler) validate() error {
	audioMarkerFilter := qb.audioMarkerFilter
	if audioMarkerFilter == nil {
		return nil
	}

	if err := validateFilterCombination(audioMarkerFilter.OperatorFilter); err != nil {
		return err
	}

	if subFilter := audioMarkerFilter.SubFilter(); subFilter != nil {
		sqb := &audioMarkerFilterHandler{audioMarkerFilter: subFilter}
		if err := sqb.validate(); err != nil {
			return err
		}
	}

	return nil
}

func (qb *audioMarkerFilterHandler) handle(ctx context.Context, f *filterBuilder) {
	audioMarkerFilter := qb.audioMarkerFilter
	if audioMarkerFilter == nil {
		return
	}

	if err := qb.validate(); err != nil {
		f.setError(err)
		return
	}

	sf := audioMarkerFilter.SubFilter()
	if sf != nil {
		sub := &audioMarkerFilterHandler{sf}
		handleSubFilter(ctx, sub, f, audioMarkerFilter.OperatorFilter)
	}

	f.handleCriterion(ctx, qb.criterionHandler())
}

func (qb *audioMarkerFilterHandler) joinAudios(f *filterBuilder) {
	audioMarkerRepository.audios.innerJoin(f, "", "audio_markers.audio_id")
}

func (qb *audioMarkerFilterHandler) criterionHandler() criterionHandler {
	audioMarkerFilter := qb.audioMarkerFilter
	return compoundHandler{
		intCriterionHandler(audioMarkerFilter.ID, "audio_markers.id", nil),
		stringCriterionHandler(audioMarkerFilter.Title, "audio_markers.title"),
		floatCriterionHandler(audioMarkerFilter.Seconds, "audio_markers.seconds", nil),
		floatCriterionHandler(audioMarkerFilter.Duration, "COALESCE(audio_markers.end_seconds - audio_markers.seconds, NULL)", nil),
		qb.audiosCriterionHandler(audioMarkerFilter.Audios),
		qb.tagsCriterionHandler(audioMarkerFilter.Tags),
		qb.audioTagsCriterionHandler(audioMarkerFilter.AudioTags),
		qb.tagCountCriterionHandler(audioMarkerFilter.TagCount),
		&timestampCriterionHandler{audioMarkerFilter.CreatedAt, "audio_markers.created_at", nil},
		&timestampCriterionHandler{audioMarkerFilter.UpdatedAt, "audio_markers.updated_at", nil},

		&relatedFilterHandler{
			relatedIDCol:   "audios.id",
			relatedRepo:    audioRepository.repository,
			relatedHandler: &audioFilterHandler{audioMarkerFilter.AudiosFilter},
			joinFn: func(f *filterBuilder) {
				qb.joinAudios(f)
			},
		},

		&relatedFilterHandler{
			relatedIDCol:   "audio_markers_tags.tag_id",
			relatedRepo:    tagRepository.repository,
			relatedHandler: &tagFilterHandler{audioMarkerFilter.TagsFilter},
			joinFn: func(f *filterBuilder) {
				f.addLeftJoin("audio_markers_tags", "", "audio_markers_tags.audio_marker_id = audio_markers.id")
			},
		},
	}
}

func (qb *audioMarkerFilterHandler) tagsCriterionHandler(criterion *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if criterion != nil {
			tags := criterion.CombineExcludes()

			if tags.Modifier == models.CriterionModifierIsNull || tags.Modifier == models.CriterionModifierNotNull {
				var notClause string
				if tags.Modifier == models.CriterionModifierNotNull {
					notClause = "NOT"
				}

				f.addLeftJoin("audio_markers_tags", "", "audio_markers.id = audio_markers_tags.audio_marker_id")

				f.addWhere(fmt.Sprintf("%s audio_markers_tags.tag_id IS NULL", notClause))
				return
			}

			if tags.Modifier == models.CriterionModifierEquals && tags.Depth != nil && *tags.Depth != 0 {
				f.setError(fmt.Errorf("depth is not supported for equals modifier for marker tag filtering"))
				return
			}

			if len(tags.Value) == 0 && len(tags.Excludes) == 0 {
				return
			}

			if len(tags.Value) > 0 {
				valuesClause, err := getHierarchicalValues(ctx, tags.Value, tagTable, "tags_relations", "parent_id", "child_id", tags.Depth)
				if err != nil {
					f.setError(err)
					return
				}

				f.addWith(`audio_marker_tags AS (
	SELECT mt.audio_marker_id, t.column1 AS root_tag_id FROM audio_markers_tags mt
	INNER JOIN (` + valuesClause + `) t ON t.column2 = mt.tag_id
	UNION
	SELECT m.id, t.column1 FROM audio_markers m
	INNER JOIN (` + valuesClause + `) t ON t.column2 = m.primary_tag_id
	)`)

				f.addLeftJoin("audio_marker_tags", "", "audio_marker_tags.audio_marker_id = audio_markers.id")

				switch tags.Modifier {
				case models.CriterionModifierEquals:
					// includes only the provided ids
					f.addWhere("audio_marker_tags.root_tag_id IS NOT NULL")
					tagsLen := len(tags.Value)
					f.addHaving(fmt.Sprintf("count(distinct audio_marker_tags.root_tag_id) IS %d", tagsLen))
					// decrement by one to account for primary tag id
					f.addWhere("(SELECT COUNT(*) FROM audio_markers_tags s WHERE s.audio_marker_id = audio_markers.id) = ?", tagsLen-1)
				case models.CriterionModifierNotEquals:
					f.setError(fmt.Errorf("not equals modifier is not supported for audio marker tags"))
				default:
					addHierarchicalConditionClauses(f, tags, "audio_marker_tags", "root_tag_id")
				}
			}

			if len(criterion.Excludes) > 0 {
				valuesClause, err := getHierarchicalValues(ctx, tags.Excludes, tagTable, "tags_relations", "parent_id", "child_id", tags.Depth)
				if err != nil {
					f.setError(err)
					return
				}

				clause := "audio_markers.id NOT IN (SELECT audio_markers_tags.audio_marker_id FROM audio_markers_tags WHERE audio_markers_tags.tag_id IN (SELECT column2 FROM (%s)))"
				f.addWhere(fmt.Sprintf(clause, valuesClause))

				f.addWhere(fmt.Sprintf("audio_markers.primary_tag_id NOT IN (SELECT column2 FROM (%s))", valuesClause))
			}
		}
	}
}

func (qb *audioMarkerFilterHandler) audiosCriterionHandler(audios *models.MultiCriterionInput) criterionHandlerFunc {
	addJoinsFunc := func(f *filterBuilder, joinType joinType) {
		f.addJoin(joinType, audioTable, "markers_audios", "markers_audios.id = audio_markers.audio_id")
	}
	h := multiCriterionHandlerBuilder{
		primaryTable: audioMarkerTable,
		foreignTable: "markers_audios",
		joinTable:    "",
		primaryFK:    audioIDColumn,
		foreignFK:    audioIDColumn,
		addJoinsFunc: addJoinsFunc,
	}
	return h.handler(audios)
}

func (qb *audioMarkerFilterHandler) tagCountCriterionHandler(tagCount *models.IntCriterionInput) criterionHandlerFunc {
	h := countCriterionHandlerBuilder{
		primaryTable: "audio_markers",
		joinTable:    "audio_markers_tags",
		primaryFK:    "audio_marker_id",
	}

	return h.handler(tagCount)
}

func (qb *audioMarkerFilterHandler) audioTagsCriterionHandler(tags *models.HierarchicalMultiCriterionInput) criterionHandlerFunc {
	return func(ctx context.Context, f *filterBuilder) {
		if tags != nil {
			f.addLeftJoin(audioTagsTable, "", "audio_markers.audio_id = audio_tags.audio_id")

			h := joinedHierarchicalMultiCriterionHandlerBuilder{
				primaryTable: "audio_markers",
				primaryKey:   audioIDColumn,
				foreignTable: tagTable,
				foreignFK:    tagIDColumn,

				relationsTable: "tags_relations",
				joinTable:      audioTagsTable,
				joinAs:         "marker_audio_tags",
				primaryFK:      audioIDColumn,
			}

			h.handler(tags).handle(ctx, f)
		}
	}
}
