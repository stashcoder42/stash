package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/logger"
	"github.com/stashapp/stash/pkg/models"
	"github.com/stashapp/stash/pkg/utils"
)

type AudioFinder interface {
	models.AudioGetter
	FindByChecksum(ctx context.Context, checksum string) ([]*models.Audio, error)
	GetCover(ctx context.Context, audioID int) ([]byte, error)
}

type audioRoutes struct {
	routes
	audioFinder       AudioFinder
	audioMarkerFinder models.AudioMarkerFinder
	fileGetter        models.FileGetter
}

func (rs audioRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{audioId}", func(r chi.Router) {
		r.Use(rs.AudioCtx)

		// streaming endpoint
		r.Get("/stream", rs.StreamDirect)
		r.Get("/cover", rs.Cover)
		r.Get("/thumbnail", rs.Thumbnail)

		// audio marker routes
		r.Get("/audio_marker/{audioMarkerId}/stream", rs.AudioMarkerStream)
		r.Get("/audio_marker/{audioMarkerId}/preview", rs.AudioMarkerPreview)
	})

	return r
}

func (rs audioRoutes) StreamDirect(w http.ResponseWriter, r *http.Request) {
	audio := r.Context().Value(audioKey).(*models.Audio)
	as := manager.AudioServer{
		TxnManager: rs.txnManager,
	}
	as.StreamAudioDirect(audio, w, r)
}

func (rs audioRoutes) Cover(w http.ResponseWriter, r *http.Request) {
	audio := r.Context().Value(audioKey).(*models.Audio)

	var cover []byte
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		cover, err = rs.audioFinder.GetCover(ctx, audio.ID)
		return err
	})
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch audio cover: %v", readTxnErr)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	if cover == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	utils.ServeImage(w, r, cover)
}

// Thumbnail serves the audio's waveform/thumbnail image
// For audio, the thumbnail is the same as the cover (waveform visualization)
func (rs audioRoutes) Thumbnail(w http.ResponseWriter, r *http.Request) {
	audio := r.Context().Value(audioKey).(*models.Audio)

	var cover []byte
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		cover, err = rs.audioFinder.GetCover(ctx, audio.ID)
		return err
	})
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch audio thumbnail: %v", readTxnErr)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	if cover == nil {
		// Return a placeholder or 404
		http.Error(w, http.StatusText(404), 404)
		return
	}

	utils.ServeImage(w, r, cover)
}

func (rs audioRoutes) AudioCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		audioID, err := strconv.Atoi(chi.URLParam(r, "audioId"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var audio *models.Audio
		_ = rs.withReadTxn(r, func(ctx context.Context) error {
			audio, _ = rs.audioFinder.Find(ctx, audioID)

			if audio != nil {
				if err := audio.LoadPrimaryFile(ctx, rs.fileGetter); err != nil {
					audio = nil
				}
			}

			return nil
		})

		if audio == nil {
			http.Error(w, http.StatusText(404), 404)
			return
		}

		// set path and checksum from primary file
		primaryFile := audio.Files.Primary()
		if primaryFile != nil {
			audio.Path = primaryFile.Base().Path
			audio.Checksum = primaryFile.Base().Fingerprints.GetString(models.FingerprintTypeOshash)
		}

		ctx := context.WithValue(r.Context(), audioKey, audio)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var audioKey = &contextKey{"audio"}

// AudioMarkerStream serves the audio stream starting at the marker's timestamp
func (rs audioRoutes) AudioMarkerStream(w http.ResponseWriter, r *http.Request) {
	audio := r.Context().Value(audioKey).(*models.Audio)
	audioMarkerID, _ := strconv.Atoi(chi.URLParam(r, "audioMarkerId"))

	var audioMarker *models.AudioMarker
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		audioMarker, err = rs.audioMarkerFinder.Find(ctx, audioMarkerID)
		return err
	})
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch audio marker: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if audioMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	// Stream the audio from the marker's timestamp
	as := manager.AudioServer{
		TxnManager: rs.txnManager,
	}
	as.StreamAudioDirect(audio, w, r)
}

// AudioMarkerPreview serves the audio's cover/thumbnail as the marker preview
// For audio markers, we use the audio's waveform as the preview since there's
// no frame-based preview like video markers
func (rs audioRoutes) AudioMarkerPreview(w http.ResponseWriter, r *http.Request) {
	audio := r.Context().Value(audioKey).(*models.Audio)
	audioMarkerID, _ := strconv.Atoi(chi.URLParam(r, "audioMarkerId"))

	var audioMarker *models.AudioMarker
	var cover []byte
	readTxnErr := rs.withReadTxn(r, func(ctx context.Context) error {
		var err error
		audioMarker, err = rs.audioMarkerFinder.Find(ctx, audioMarkerID)
		if err != nil {
			return err
		}
		cover, err = rs.audioFinder.GetCover(ctx, audio.ID)
		return err
	})
	if readTxnErr != nil {
		logger.Warnf("read transaction error on fetch audio marker preview: %v", readTxnErr)
		http.Error(w, readTxnErr.Error(), http.StatusInternalServerError)
		return
	}

	if audioMarker == nil {
		http.Error(w, http.StatusText(404), 404)
		return
	}

	if cover == nil {
		// Return placeholder if no cover exists
		w.Header().Set("Content-Type", "image/png")
		utils.ServeStaticContent(w, r, utils.PendingGenerateResource)
		return
	}

	utils.ServeImage(w, r, cover)
}
