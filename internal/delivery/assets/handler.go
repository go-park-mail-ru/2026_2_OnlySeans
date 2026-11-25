package assets

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

type Handler struct {
	client     *s3.Client
	bucket     string
	appLogger  logger.Logger
	operations storage.OperationRecorder
}

func NewHandler(client *s3.Client, bucket string, appLogger logger.Logger, observers ...storage.ExternalObserver) *Handler {
	return &Handler{
		client: client, bucket: bucket, appLogger: appLogger,
		operations: storage.NewOperationRecorder(appLogger, observers...),
	}
}

func (h *Handler) GetAsset(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	if key == "" || strings.Contains(key, "..") {
		http.NotFound(w, r)
		return
	}

	operation := h.operations.Start(r.Context(), "s3", "asset.get", map[string]any{"bucket": h.bucket, "storage_key": key})
	object, err := h.client.GetObject(r.Context(), &s3.GetObjectInput{Bucket: aws.String(h.bucket), Key: aws.String(key)})
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		err = domain.ErrFileNotFound
	}
	operation.Finish(err)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer object.Body.Close()

	if object.ContentType != nil {
		w.Header().Set("Content-Type", *object.ContentType)
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	if _, err := io.Copy(w, object.Body); err != nil && h.appLogger != nil {
		h.appLogger.ErrorContext(r.Context(), "write asset failed", map[string]any{"error": err})
	}
}
