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
	audioFinder AudioFinder
	fileGetter  models.FileGetter
}

func (rs audioRoutes) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{audioId}", func(r chi.Router) {
		r.Use(rs.AudioCtx)

		// streaming endpoint
		r.Get("/stream", rs.StreamDirect)
		r.Get("/cover", rs.Cover)
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
