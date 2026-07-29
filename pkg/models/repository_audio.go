package models

import (
	"context"
	"time"
)

// AudioGetter provides methods to get audio by ID.
type AudioGetter interface {
	FindMany(ctx context.Context, ids []int) ([]*Audio, error)
	Find(ctx context.Context, id int) (*Audio, error)
}

// AudioFinder provides methods to find audio.
type AudioFinder interface {
	AudioGetter
	FindByChecksum(ctx context.Context, checksum string) ([]*Audio, error)
	FindByPath(ctx context.Context, path string) (*Audio, error)
	FindByFileID(ctx context.Context, fileID FileID) ([]*Audio, error)
	FindByFingerprints(ctx context.Context, fp []Fingerprint) ([]*Audio, error)
	FindByPrimaryFileID(ctx context.Context, fileID FileID) ([]*Audio, error)
	FindByPerformerID(ctx context.Context, performerID int) ([]*Audio, error)
}

// AudioQueryer provides methods to query audio.
type AudioQueryer interface {
	Query(ctx context.Context, options AudioQueryOptions) (*AudioQueryResult, error)
	QueryCount(ctx context.Context, audioFilter *AudioFilterType, findFilter *FindFilterType) (int, error)
}

// AudioCounter provides methods to count audio.
type AudioCounter interface {
	Count(ctx context.Context) (int, error)
	CountByPerformerID(ctx context.Context, performerID int) (int, error)
	CountByFileID(ctx context.Context, fileID FileID) (int, error)
	OCount(ctx context.Context) (int, error)
	OCountByPerformerID(ctx context.Context, performerID int) (int, error)
}

// AudioCreator provides methods to create audio.
type AudioCreator interface {
	Create(ctx context.Context, newAudio *Audio, fileIDs []FileID) error
}

// AudioUpdater provides methods to update audio.
type AudioUpdater interface {
	Update(ctx context.Context, updatedAudio *Audio) error
	UpdatePartial(ctx context.Context, id int, updatedAudio AudioPartial) (*Audio, error)
}

// AudioDestroyer provides methods to destroy audio.
type AudioDestroyer interface {
	Destroy(ctx context.Context, id int) error
}

type AudioCreatorUpdater interface {
	AudioCreator
	AudioUpdater
}

// AudioViewDateReader provides methods to read audio view history.
type AudioViewDateReader interface {
	CountViews(ctx context.Context, id int) (int, error)
	CountAllViews(ctx context.Context) (int, error)
	CountUniqueViews(ctx context.Context) (int, error)
	GetManyViewCount(ctx context.Context, ids []int) ([]int, error)
	GetViewDates(ctx context.Context, relatedID int) ([]time.Time, error)
	GetManyViewDates(ctx context.Context, ids []int) ([][]time.Time, error)
	GetManyLastViewed(ctx context.Context, ids []int) ([]*time.Time, error)
	LastView(ctx context.Context, id int) (*time.Time, error)
}

// AudioODateReader provides methods to read audio o-counter history.
type AudioODateReader interface {
	GetOCount(ctx context.Context, id int) (int, error)
	GetManyOCount(ctx context.Context, ids []int) ([]int, error)
	GetAllOCount(ctx context.Context) (int, error)
	GetUniqueOCount(ctx context.Context) (int, error)
	GetODates(ctx context.Context, relatedID int) ([]time.Time, error)
	GetManyODates(ctx context.Context, ids []int) ([][]time.Time, error)
}

// AudioReader provides all methods to read audio.
type AudioReader interface {
	AudioFinder
	AudioQueryer
	AudioCounter

	FileIDLoader
	PerformerIDLoader
	TagIDLoader
	FileLoader
	URLLoader
	AudioViewDateReader
	AudioODateReader
	CustomFieldsReader

	All(ctx context.Context) ([]*Audio, error)
	Size(ctx context.Context) (float64, error)
	Duration(ctx context.Context) (float64, error)
}

// AudioOHistoryWriter provides methods to modify audio o-counter history.
type AudioOHistoryWriter interface {
	AddO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error)
	DeleteO(ctx context.Context, id int, dates []time.Time) ([]time.Time, error)
	ResetO(ctx context.Context, id int) (int, error)
}

// AudioViewHistoryWriter provides methods to modify audio view history.
type AudioViewHistoryWriter interface {
	AddViews(ctx context.Context, audioID int, dates []time.Time) ([]time.Time, error)
	DeleteViews(ctx context.Context, id int, dates []time.Time) ([]time.Time, error)
	DeleteAllViews(ctx context.Context, id int) (int, error)
}

// AudioWriter provides all methods to modify audio.
type AudioWriter interface {
	AudioCreator
	AudioUpdater
	AudioDestroyer
	CustomFieldsWriter

	AddFileID(ctx context.Context, id int, fileID FileID) error
	AssignFiles(ctx context.Context, audioID int, fileID []FileID) error

	AudioOHistoryWriter
	AudioViewHistoryWriter
	SaveActivity(ctx context.Context, audioID int, resumeTime *float64, playDuration *float64) (bool, error)
	ResetActivity(ctx context.Context, audioID int, resetResume bool, resetDuration bool) (bool, error)

	// Cover image methods
	GetCover(ctx context.Context, audioID int) ([]byte, error)
	UpdateCover(ctx context.Context, audioID int, image []byte) error
	HasCover(ctx context.Context, audioID int) (bool, error)
}

// AudioReaderWriter provides all audio methods.
type AudioReaderWriter interface {
	AudioReader
	AudioWriter
}
