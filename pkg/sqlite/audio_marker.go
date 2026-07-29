package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/jmoiron/sqlx"
	"gopkg.in/guregu/null.v4"

	"github.com/stashapp/stash/pkg/models"
)

const audioMarkerTable = "audio_markers"

const countAudioMarkersForTagQuery = `
SELECT audio_markers.id FROM audio_markers
LEFT JOIN audio_markers_tags as tags_join on tags_join.audio_marker_id = audio_markers.id
WHERE tags_join.tag_id = ? OR audio_markers.primary_tag_id = ?
GROUP BY audio_markers.id
`

type audioMarkerRow struct {
	ID           int        `db:"id" goqu:"skipinsert"`
	Title        string     `db:"title"` // TODO: make db schema (and gql schema) nullable
	Seconds      float64    `db:"seconds"`
	PrimaryTagID int        `db:"primary_tag_id"`
	AudioID      int        `db:"audio_id"`
	CreatedAt    Timestamp  `db:"created_at"`
	UpdatedAt    Timestamp  `db:"updated_at"`
	EndSeconds   null.Float `db:"end_seconds"`
}

func (r *audioMarkerRow) fromAudioMarker(o models.AudioMarker) {
	r.ID = o.ID
	r.Title = o.Title
	r.Seconds = o.Seconds
	if o.EndSeconds != nil {
		r.EndSeconds = null.FloatFrom(*o.EndSeconds)
	}
	r.PrimaryTagID = o.PrimaryTagID
	r.AudioID = o.AudioID
	r.CreatedAt = Timestamp{Timestamp: o.CreatedAt}
	r.UpdatedAt = Timestamp{Timestamp: o.UpdatedAt}
}

func (r *audioMarkerRow) resolve() *models.AudioMarker {
	ret := &models.AudioMarker{
		ID:           r.ID,
		Title:        r.Title,
		Seconds:      r.Seconds,
		EndSeconds:   r.EndSeconds.Ptr(),
		PrimaryTagID: r.PrimaryTagID,
		AudioID:      r.AudioID,
		CreatedAt:    r.CreatedAt.Timestamp,
		UpdatedAt:    r.UpdatedAt.Timestamp,
	}

	return ret
}

type audioMarkerRowRecord struct {
	updateRecord
}

func (r *audioMarkerRowRecord) fromPartial(o models.AudioMarkerPartial) {
	// TODO: replace with setNullString after schema is made nullable
	// r.setNullString("title", o.Title)
	// saves a null input as the empty string
	if o.Title.Set {
		r.set("title", o.Title.Value)
	}
	r.setFloat64("seconds", o.Seconds)
	r.setNullFloat64("end_seconds", o.EndSeconds)
	r.setInt("primary_tag_id", o.PrimaryTagID)
	r.setInt("audio_id", o.AudioID)
	r.setTimestamp("created_at", o.CreatedAt)
	r.setTimestamp("updated_at", o.UpdatedAt)
}

type audioMarkerRepositoryType struct {
	repository

	audios repository
	tags   joinRepository
}

var (
	audioMarkerRepository = audioMarkerRepositoryType{
		repository: repository{
			tableName: audioMarkerTable,
			idColumn:  idColumn,
		},
		audios: repository{
			tableName: audioTable,
			idColumn:  idColumn,
		},
		tags: joinRepository{
			repository: repository{
				tableName: "audio_markers_tags",
				idColumn:  "audio_marker_id",
			},
			fkColumn: tagIDColumn,
		},
	}
)

type AudioMarkerStore struct{}

func NewAudioMarkerStore() *AudioMarkerStore {
	return &AudioMarkerStore{}
}

func (qb *AudioMarkerStore) table() exp.IdentifierExpression {
	return audioMarkerTableMgr.table
}

func (qb *AudioMarkerStore) selectDataset() *goqu.SelectDataset {
	return dialect.From(qb.table()).Select(qb.table().All())
}

func (qb *AudioMarkerStore) Create(ctx context.Context, newObject *models.AudioMarker) error {
	var r audioMarkerRow
	r.fromAudioMarker(*newObject)

	id, err := audioMarkerTableMgr.insertID(ctx, r)
	if err != nil {
		return err
	}

	updated, err := qb.find(ctx, id)
	if err != nil {
		return fmt.Errorf("finding after create: %w", err)
	}

	*newObject = *updated

	return nil
}

func (qb *AudioMarkerStore) UpdatePartial(ctx context.Context, id int, partial models.AudioMarkerPartial) (*models.AudioMarker, error) {
	r := audioMarkerRowRecord{
		updateRecord{
			Record: make(exp.Record),
		},
	}

	r.fromPartial(partial)

	if len(r.Record) > 0 {
		if err := audioMarkerTableMgr.updateByID(ctx, id, r.Record); err != nil {
			return nil, err
		}
	}

	return qb.find(ctx, id)
}

func (qb *AudioMarkerStore) Update(ctx context.Context, updatedObject *models.AudioMarker) error {
	var r audioMarkerRow
	r.fromAudioMarker(*updatedObject)

	if err := audioMarkerTableMgr.updateByID(ctx, updatedObject.ID, r); err != nil {
		return err
	}

	return nil
}

func (qb *AudioMarkerStore) Destroy(ctx context.Context, id int) error {
	return audioMarkerRepository.destroyExisting(ctx, []int{id})
}

// returns nil, nil if not found
func (qb *AudioMarkerStore) Find(ctx context.Context, id int) (*models.AudioMarker, error) {
	ret, err := qb.find(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ret, err
}

func (qb *AudioMarkerStore) FindMany(ctx context.Context, ids []int) ([]*models.AudioMarker, error) {
	ret := make([]*models.AudioMarker, len(ids))

	table := qb.table()
	q := qb.selectDataset().Prepared(true).Where(table.Col(idColumn).In(ids))
	unsorted, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	for _, s := range unsorted {
		i := slices.Index(ids, s.ID)
		ret[i] = s
	}

	for i := range ret {
		if ret[i] == nil {
			return nil, fmt.Errorf("audio marker with id %d not found", ids[i])
		}
	}

	return ret, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *AudioMarkerStore) find(ctx context.Context, id int) (*models.AudioMarker, error) {
	q := qb.selectDataset().Where(audioMarkerTableMgr.byID(id))

	ret, err := qb.get(ctx, q)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// returns nil, sql.ErrNoRows if not found
func (qb *AudioMarkerStore) get(ctx context.Context, q *goqu.SelectDataset) (*models.AudioMarker, error) {
	ret, err := qb.getMany(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, sql.ErrNoRows
	}

	return ret[0], nil
}

func (qb *AudioMarkerStore) getMany(ctx context.Context, q *goqu.SelectDataset) ([]*models.AudioMarker, error) {
	const single = false
	var ret []*models.AudioMarker
	if err := queryFunc(ctx, q, single, func(r *sqlx.Rows) error {
		var f audioMarkerRow
		if err := r.StructScan(&f); err != nil {
			return err
		}

		s := f.resolve()

		ret = append(ret, s)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *AudioMarkerStore) FindByAudioID(ctx context.Context, audioID int) ([]*models.AudioMarker, error) {
	query := `
		SELECT audio_markers.* FROM audio_markers
		WHERE audio_markers.audio_id = ?
		GROUP BY audio_markers.id
		ORDER BY audio_markers.seconds ASC
	`
	args := []interface{}{audioID}
	return qb.queryAudioMarkers(ctx, query, args)
}

func (qb *AudioMarkerStore) CountByTagID(ctx context.Context, tagID int) (int, error) {
	args := []interface{}{tagID, tagID}
	return audioMarkerRepository.runCountQuery(ctx, audioMarkerRepository.buildCountQuery(countAudioMarkersForTagQuery), args)
}

func (qb *AudioMarkerStore) GetMarkerStrings(ctx context.Context, q *string, sort *string) ([]*models.MarkerStringsResultType, error) {
	query := "SELECT count(*) as `count`, audio_markers.id as id, audio_markers.title as title FROM audio_markers"
	if q != nil {
		query += " WHERE title LIKE '%" + *q + "%'"
	}
	query += " GROUP BY title"
	if sort != nil && *sort == "count" {
		query += " ORDER BY `count` DESC"
	} else {
		query += " ORDER BY title ASC"
	}
	var args []interface{}
	return qb.queryMarkerStringsResultType(ctx, query, args)
}

func (qb *AudioMarkerStore) Wall(ctx context.Context, q *string) ([]*models.AudioMarker, error) {
	s := ""
	if q != nil {
		s = *q
	}

	table := qb.table()
	qq := qb.selectDataset().Prepared(true).Where(table.Col("title").Like("%" + s + "%")).Order(goqu.L("RANDOM()").Asc()).Limit(80)
	return qb.getMany(ctx, qq)
}

func (qb *AudioMarkerStore) makeQuery(ctx context.Context, audioMarkerFilter *models.AudioMarkerFilterType, findFilter *models.FindFilterType) (*queryBuilder, error) {
	if audioMarkerFilter == nil {
		audioMarkerFilter = &models.AudioMarkerFilterType{}
	}
	if findFilter == nil {
		findFilter = &models.FindFilterType{}
	}

	query := audioMarkerRepository.newQuery()
	distinctIDs(&query, audioMarkerTable)

	if q := findFilter.Q; q != nil && *q != "" {
		query.join(audioTable, "", "audios.id = audio_markers.audio_id")
		query.join(tagTable, "", "audio_markers.primary_tag_id = tags.id")
		searchColumns := []string{"audio_markers.title", "audios.title", "tags.name"}
		query.parseQueryString(searchColumns, *q)
	}

	filter := filterBuilderFromHandler(ctx, &audioMarkerFilterHandler{
		audioMarkerFilter: audioMarkerFilter,
	})

	if err := query.addFilter(filter); err != nil {
		return nil, err
	}

	if err := qb.setAudioMarkerSort(&query, findFilter); err != nil {
		return nil, err
	}
	query.sortAndPagination += getPagination(findFilter)

	return &query, nil
}

func (qb *AudioMarkerStore) Query(ctx context.Context, audioMarkerFilter *models.AudioMarkerFilterType, findFilter *models.FindFilterType) ([]*models.AudioMarker, int, error) {
	query, err := qb.makeQuery(ctx, audioMarkerFilter, findFilter)
	if err != nil {
		return nil, 0, err
	}

	idsResult, countResult, err := query.executeFind(ctx)
	if err != nil {
		return nil, 0, err
	}

	audioMarkers, err := qb.FindMany(ctx, idsResult)
	if err != nil {
		return nil, 0, err
	}

	return audioMarkers, countResult, nil
}

func (qb *AudioMarkerStore) QueryCount(ctx context.Context, audioMarkerFilter *models.AudioMarkerFilterType, findFilter *models.FindFilterType) (int, error) {
	query, err := qb.makeQuery(ctx, audioMarkerFilter, findFilter)
	if err != nil {
		return 0, err
	}

	return query.executeCount(ctx)
}

var audioMarkerSortOptions = sortOptions{
	"created_at",
	"id",
	"title",
	"random",
	"audio_id",
	"audios_updated_at",
	"seconds",
	"updated_at",
	"duration",
}

func (qb *AudioMarkerStore) setAudioMarkerSort(query *queryBuilder, findFilter *models.FindFilterType) error {
	sort := findFilter.GetSort("title")
	direction := findFilter.GetDirection()

	// CVE-2024-32231 - ensure sort is in the list of allowed sorts
	if err := audioMarkerSortOptions.validateSort(sort); err != nil {
		return err
	}

	switch sort {
	case "audios_updated_at":
		sort = "updated_at"
		query.join(audioTable, "", "audios.id = audio_markers.audio_id")
		query.sortAndPagination += getSort(sort, direction, audioTable)
	case "title":
		query.join(tagTable, "", "audio_markers.primary_tag_id = tags.id")
		query.sortAndPagination += " ORDER BY COALESCE(NULLIF(audio_markers.title,''), tags.name) COLLATE NATURAL_CI " + direction
	case "duration":
		sort = "(audio_markers.end_seconds - audio_markers.seconds)"
		query.sortAndPagination += getSort(sort, direction, audioMarkerTable)
	default:
		query.sortAndPagination += getSort(sort, direction, audioMarkerTable)
	}

	query.sortAndPagination += ", audio_markers.audio_id ASC, audio_markers.seconds ASC"
	return nil
}

func (qb *AudioMarkerStore) queryAudioMarkers(ctx context.Context, query string, args []interface{}) ([]*models.AudioMarker, error) {
	var ret []*models.AudioMarker
	if err := audioMarkerRepository.queryFunc(ctx, query, args, true, func(rows *sqlx.Rows) error {
		var f audioMarkerRow
		if err := rows.StructScan(&f); err != nil {
			return err
		}

		s := f.resolve()

		ret = append(ret, s)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *AudioMarkerStore) queryMarkerStringsResultType(ctx context.Context, query string, args []interface{}) ([]*models.MarkerStringsResultType, error) {
	var ret []*models.MarkerStringsResultType
	if err := audioMarkerRepository.queryFunc(ctx, query, args, true, func(rows *sqlx.Rows) error {
		var f models.MarkerStringsResultType
		if err := rows.StructScan(&f); err != nil {
			return err
		}

		ret = append(ret, &f)
		return nil
	}); err != nil {
		return nil, err
	}

	return ret, nil
}

func (qb *AudioMarkerStore) GetTagIDs(ctx context.Context, id int) ([]int, error) {
	return audioMarkerRepository.tags.getIDs(ctx, id)
}

func (qb *AudioMarkerStore) UpdateTags(ctx context.Context, id int, tagIDs []int) error {
	return audioMarkerRepository.tags.replace(ctx, id, tagIDs)
}

func (qb *AudioMarkerStore) Count(ctx context.Context) (int, error) {
	q := dialect.Select(goqu.COUNT("*")).From(qb.table())
	return count(ctx, q)
}

func (qb *AudioMarkerStore) All(ctx context.Context) ([]*models.AudioMarker, error) {
	return qb.getMany(ctx, qb.selectDataset())
}
