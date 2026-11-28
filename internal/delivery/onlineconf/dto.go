package onlineconf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
)

const (
	MaxRequestBodyBytes       = 256 * 1024
	resourceIDParameter       = "id"
	scopeParameter            = "scope"
	queryParameter            = "q"
	limitParameter            = "limit"
	offsetParameter           = "offset"
	versionParameter          = "version"
	resourceTypeParameter     = "resource_type"
	resourceIDFilterParameter = "resource_id"
	decimalBase               = 10
	integerBits               = 64
)

type scopeRequest struct {
	Version     *int64  `json:"version"`
	Name        *string `json:"name"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

type configRequest struct {
	Scope       *string         `json:"scope"`
	Key         *string         `json:"key"`
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	Value       json.RawMessage `json:"value"`
	Version     *int64          `json:"version"`
}

type flagRequest struct {
	Scope           *string          `json:"scope"`
	Key             *string          `json:"key"`
	Title           *string          `json:"title"`
	Description     *string          `json:"description"`
	Enabled         *bool            `json:"enabled"`
	RolloutPercent  *int             `json:"rollout_percent"`
	ForceOnUserIDs  *[]domain.UserID `json:"force_on_user_ids"`
	ForceOffUserIDs *[]domain.UserID `json:"force_off_user_ids"`
	Version         *int64           `json:"version"`
}

type evaluateRequest struct {
	UserID *domain.UserID `json:"user_id"`
}

type flagResponse struct {
	ID              int64           `json:"id"`
	Scope           string          `json:"scope"`
	Key             string          `json:"key"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Enabled         bool            `json:"enabled"`
	RolloutPercent  int             `json:"rollout_percent"`
	ForceOnUserIDs  []domain.UserID `json:"force_on_user_ids"`
	ForceOffUserIDs []domain.UserID `json:"force_off_user_ids"`
	Version         int64           `json:"version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func publicFlag(flag *oc.FeatureFlag) flagResponse {
	return flagResponse{ID: flag.ID, Scope: flag.Scope, Key: flag.Key, Title: flag.Title, Description: flag.Description, Enabled: flag.Enabled, RolloutPercent: flag.RolloutPercent, ForceOnUserIDs: append([]domain.UserID{}, flag.ForceOnUserIDs...), ForceOffUserIDs: append([]domain.UserID{}, flag.ForceOffUserIDs...), Version: flag.Version, CreatedAt: flag.CreatedAt.UTC(), UpdatedAt: flag.UpdatedAt.UTC()}
}

func decodeRequest[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request *T

	if err := decoder.Decode(&request); err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidJSON, err)
	}

	if request == nil {
		return nil, errInvalidJSON
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errInvalidJSON
	}

	return request, nil
}

func (req configRequest) fields(update bool) (usecase.ConfigFields, error) {
	if req.Scope == nil || req.Key == nil || req.Title == nil || len(req.Value) == 0 || (!update && req.Version != nil) || (update && (req.Version == nil || req.Description == nil)) {
		return usecase.ConfigFields{}, errInvalidInput
	}

	return usecase.ConfigFields{Scope: *req.Scope, Key: *req.Key, Title: *req.Title, Description: optionalString(req.Description), Value: req.Value}, nil
}

func (req flagRequest) fields(update bool) (usecase.FlagFields, error) {
	if req.Scope == nil || req.Key == nil || req.Title == nil || req.Enabled == nil || req.RolloutPercent == nil || req.ForceOnUserIDs == nil || req.ForceOffUserIDs == nil || (!update && req.Version != nil) || (update && (req.Version == nil || req.Description == nil)) {
		return usecase.FlagFields{}, errInvalidInput
	}

	return usecase.FlagFields{Scope: *req.Scope, Key: *req.Key, Title: *req.Title, Description: optionalString(req.Description), Enabled: *req.Enabled, RolloutPercent: *req.RolloutPercent, ForceOnUserIDs: *req.ForceOnUserIDs, ForceOffUserIDs: *req.ForceOffUserIDs}, nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, decimalBase, integerBits)
	if err != nil || id < oc.MinUserID || id > oc.MaxSafeID {
		return 0, oc.ErrInvalidID
	}

	return id, nil
}

func requestID(r *http.Request) (int64, error) { return parseID(mux.Vars(r)[resourceIDParameter]) }

func deleteInput(r *http.Request) (usecase.DeleteResourceInput, error) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		return usecase.DeleteResourceInput{}, domain.ErrUnauthorized
	}

	id, err := requestID(r)
	if err != nil {
		return usecase.DeleteResourceInput{}, err
	}

	version, err := strconv.ParseInt(r.URL.Query().Get(versionParameter), decimalBase, integerBits)
	if err != nil || version < 1 {
		return usecase.DeleteResourceInput{}, oc.ErrInvalidVersion
	}

	return usecase.DeleteResourceInput{ActorID: actor.ID, ID: id, Version: version}, nil
}

func requestList(r *http.Request) (oc.ListOptions, error) {
	values := r.URL.Query()
	options := oc.ListOptions{Scope: values.Get(scopeParameter), Query: values.Get(queryParameter), Limit: oc.DefaultListLimit}

	if values.Has(limitParameter) {
		limit, err := strconv.Atoi(values.Get(limitParameter))
		if err != nil || limit < 1 || limit > oc.MaxListLimit {
			return options, oc.ErrInvalidPagination
		}

		options.Limit = limit
	}

	if values.Has(offsetParameter) {
		offset, err := strconv.Atoi(values.Get(offsetParameter))
		if err != nil || offset < 0 {
			return options, oc.ErrInvalidPagination
		}

		options.Offset = offset
	}

	return options, nil
}
