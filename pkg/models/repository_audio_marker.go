package models

import "context"

// AudioMarkerGetter provides methods to get audio markers by ID.
type AudioMarkerGetter interface {
	// TODO - rename this to Find and remove existing method
	FindMany(ctx context.Context, ids []int) ([]*AudioMarker, error)
	Find(ctx context.Context, id int) (*AudioMarker, error)
}

// AudioMarkerFinder provides methods to find audio markers.
type AudioMarkerFinder interface {
	AudioMarkerGetter
	FindByAudioID(ctx context.Context, audioID int) ([]*AudioMarker, error)
}

// AudioMarkerQueryer provides methods to query audio markers.
type AudioMarkerQueryer interface {
	Query(ctx context.Context, audioMarkerFilter *AudioMarkerFilterType, findFilter *FindFilterType) ([]*AudioMarker, int, error)
	QueryCount(ctx context.Context, audioMarkerFilter *AudioMarkerFilterType, findFilter *FindFilterType) (int, error)
}

// AudioMarkerCounter provides methods to count audio markers.
type AudioMarkerCounter interface {
	Count(ctx context.Context) (int, error)
	CountByTagID(ctx context.Context, tagID int) (int, error)
}

// AudioMarkerCreator provides methods to create audio markers.
type AudioMarkerCreator interface {
	Create(ctx context.Context, newAudioMarker *AudioMarker) error
}

// AudioMarkerUpdater provides methods to update audio markers.
type AudioMarkerUpdater interface {
	Update(ctx context.Context, updatedAudioMarker *AudioMarker) error
	UpdatePartial(ctx context.Context, id int, updatedAudioMarker AudioMarkerPartial) (*AudioMarker, error)
	UpdateTags(ctx context.Context, markerID int, tagIDs []int) error
}

// AudioMarkerDestroyer provides methods to destroy audio markers.
type AudioMarkerDestroyer interface {
	Destroy(ctx context.Context, id int) error
}

type AudioMarkerCreatorUpdater interface {
	AudioMarkerCreator
	AudioMarkerUpdater
}

// AudioMarkerReader provides all methods to read audio markers.
type AudioMarkerReader interface {
	AudioMarkerFinder
	AudioMarkerQueryer
	AudioMarkerCounter

	TagIDLoader

	All(ctx context.Context) ([]*AudioMarker, error)
	Wall(ctx context.Context, q *string) ([]*AudioMarker, error)
	GetMarkerStrings(ctx context.Context, q *string, sort *string) ([]*MarkerStringsResultType, error)
}

// AudioMarkerWriter provides all methods to modify audio markers.
type AudioMarkerWriter interface {
	AudioMarkerCreator
	AudioMarkerUpdater
	AudioMarkerDestroyer
}

// AudioMarkerReaderWriter provides all audio marker methods.
type AudioMarkerReaderWriter interface {
	AudioMarkerReader
	AudioMarkerWriter
}
