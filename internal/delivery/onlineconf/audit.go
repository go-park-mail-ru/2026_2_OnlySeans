package onlineconf

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

func (h *Handler) listAudit(w http.ResponseWriter, r *http.Request) {
	options, err := requestList(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	values := r.URL.Query()
	filter := oc.AuditFilter{ResourceKind: oc.ResourceKind(values.Get(resourceTypeParameter)), Limit: options.Limit, Offset: options.Offset}

	if values.Has(resourceIDFilterParameter) {
		id, err := parseID(values.Get(resourceIDFilterParameter))
		if err != nil {
			h.writeError(w, r, err)
			return
		}

		filter.ResourceID = id
	}

	result, err := h.audit.Run(r.Context(), filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}
