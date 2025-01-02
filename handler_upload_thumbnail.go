package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot parse form", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot get thumbnail", err)
		return
	}

	defer file.Close()

	mediaType := fileHeader.Header.Get("Content-Type")

	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}

	// imgData, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Could not read image data", err)
	// 	return
	// }

	// encodeStr := base64.StdEncoding.EncodeToString(imgData)
	// dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, encodeStr)

	newFile, err := os.Create(filepath.Join(cfg.assetsRoot, videoIDString))
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Could not create file data to asset dir",
			err,
		)
		return
	}

	io.Copy(newFile, file)

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot get video metadata", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User is not video owner", err)
		return
	}

	// thumbnailURL := fmt.Sprintf("http://localhost:%s/api/thumbnails/{%s}", cfg.port, videoID)

	staticURL := fmt.Sprintf("http://localhost:<port>/assets/%s.%s", videoID, mediaType)

	videoMetadata.ThumbnailURL = &staticURL
	videoMetadata.UpdatedAt = time.Now()
	delete(videoThumbnails, videoID)

	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Cannot upload video thumnail", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
