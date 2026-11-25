package library

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *Handler) Preferences(w http.ResponseWriter, r *http.Request) {
	result, err := h.client.GetPreferences(r.Context(), user(r))
	if err != nil {
		h.fail(r.Context(), w, err)
		return
	}

	data := make(map[string]any)
	preferences := result.GetPreferences()

	if preferences.SidebarCollapsed != nil {
		data["frame:sidebar-collapsed"] = preferences.GetSidebarCollapsed()
	}

	if preferences.CatalogView != nil {
		data["frame:catalog-view"] = preferences.GetCatalogView()
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, data, h.appLogger)
}

func (h *Handler) SavePreference(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
	}

	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		h.fail(r.Context(), w, err)
		return
	}

	request := &pb.SavePreferenceRequest{
		UserId: user(r),
	}

	switch body.Key {
	case "frame:sidebar-collapsed":
		if string(body.Value) != "true" && string(body.Value) != "false" {
			h.fail(r.Context(), w, catalogclient.ErrInvalidLibraryData)
			return
		}

		request.Update = &pb.SavePreferenceRequest_SidebarCollapsed{
			SidebarCollapsed: string(body.Value) == "true",
		}
	case "frame:catalog-view":
		if string(body.Value) != `"grid"` && string(body.Value) != `"list"` {
			h.fail(r.Context(), w, catalogclient.ErrInvalidLibraryData)
			return
		}

		var view string
		if err := json.Unmarshal(body.Value, &view); err != nil {
			h.fail(r.Context(), w, catalogclient.ErrInvalidLibraryData)
			return
		}

		request.Update = &pb.SavePreferenceRequest_CatalogView{
			CatalogView: view,
		}
	default:
		h.fail(r.Context(), w, catalogclient.ErrInvalidLibraryData)
		return
	}

	if _, err := h.client.SavePreference(r.Context(), request); err != nil {
		h.fail(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
