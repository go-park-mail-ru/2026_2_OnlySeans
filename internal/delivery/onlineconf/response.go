package onlineconf

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

const (
	invalidInputCode     = "invalid_input"
	invalidJSONCode      = "invalid_json"
	notFoundCode         = "not_found"
	versionConflictCode  = "version_conflict"
	duplicateKeyCode     = "duplicate_key"
	resourceInUseCode    = "resource_in_use"
	methodNotAllowedCode = "method_not_allowed"
	runtimeNotReadyCode  = "runtime_not_ready"
	snapshotTooLargeCode = "snapshot_too_large"
)

var (
	errInvalidInput = errors.New("invalid request parameters")
	errInvalidJSON  = errors.New("invalid JSON request body")
)

func (h *AdminHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, internalErrorCode, domain.ErrInternal.Error()
	var blockError oc.HomeBlocksError

	switch {
	case errors.As(err, &blockError):
		status, code, message = http.StatusBadRequest, invalidInputCode, blockError.Error()
	case errors.Is(err, oc.ErrSnapshotTooLarge):
		status, code, message = http.StatusRequestEntityTooLarge, snapshotTooLargeCode, oc.ErrSnapshotTooLarge.Error()
	case errors.Is(err, errInvalidJSON), errors.Is(err, oc.ErrInvalidJSON):
		status, code, message = http.StatusBadRequest, invalidJSONCode, "invalid JSON request body"
	case errors.Is(err, oc.ErrVersionConflict):
		status, code, message = http.StatusConflict, versionConflictCode, oc.ErrVersionConflict.Error()
	case errors.Is(err, oc.ErrDuplicateKey):
		status, code, message = http.StatusConflict, duplicateKeyCode, oc.ErrDuplicateKey.Error()
	case errors.Is(err, oc.ErrResourceInUse):
		status, code, message = http.StatusConflict, resourceInUseCode, oc.ErrResourceInUse.Error()
	case errors.Is(err, oc.ErrNotFound), errors.Is(err, domain.ErrUserNotFound):
		status, code, message = http.StatusNotFound, notFoundCode, oc.ErrNotFound.Error()
	case errors.Is(err, oc.ErrSnapshotNotReady):
		status, code, message = http.StatusServiceUnavailable, runtimeNotReadyCode, oc.ErrSnapshotNotReady.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		status, code, message = http.StatusUnauthorized, unauthorizedCode, domain.ErrUnauthorized.Error()
	case errors.Is(err, domain.ErrForbidden):
		status, code, message = http.StatusForbidden, forbiddenCode, domain.ErrForbidden.Error()
	default:
		for _, invalid := range []error{errInvalidInput, oc.ErrInvalidID, oc.ErrInvalidVersion, oc.ErrImmutableIdentity, oc.ErrInvalidPagination, oc.ErrInvalidAuditFilter, oc.ErrInvalidKey, oc.ErrInvalidTitle, oc.ErrValueTooLarge, oc.ErrInvalidReviewLimit, oc.ErrInvalidRollout, oc.ErrInvalidUserID, oc.ErrConflictingOverrides, oc.ErrDuplicateOverride} {
			if errors.Is(err, invalid) {
				status, code, message = http.StatusBadRequest, invalidInputCode, invalid.Error()
				break
			}
		}
	}

	if status >= http.StatusInternalServerError {
		h.appLogger.ErrorContext(r.Context(), "onlineconf API request failed", map[string]any{"path": r.URL.Path, "status": status})
	}

	httpresponse.WriteJSONContext(r.Context(), w, status, accessErrorResponse{Error: message, Code: code}, h.appLogger)
}

func (h *AdminHandler) notFound(w http.ResponseWriter, r *http.Request) {
	h.writeError(w, r, oc.ErrNotFound)
}

func (h *AdminHandler) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusMethodNotAllowed, accessErrorResponse{Error: "method not allowed", Code: methodNotAllowedCode}, h.appLogger)
}
