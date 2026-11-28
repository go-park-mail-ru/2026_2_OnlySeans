package releases_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/openapi"
)

func TestPatchModelPreservesMissingNullAndDate(t *testing.T) {
	for _, input := range []string{`{}`, `{"world_premiere_date":null}`, `{"world_premiere_date":"2026-12-10"}`} {
		var model openapi.ReleaseCalendarPatch
		require.NoError(t, json.Unmarshal([]byte(input), &model))
		encoded, err := json.Marshal(model)
		require.NoError(t, err)
		require.JSONEq(t, input, string(encoded))
	}
}
