package onlineconf_test

import (
	"encoding/json"
	"testing"

	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

func TestShowcaseWriteValidation(t *testing.T) {
	for _, c := range []struct {
		key, value string
		valid      bool
	}{
		{"collection_visibility", `{"mode":"exclude","items":[]}`, true},
		{"collection_visibility", `{"mode":"include","items":[1,1]}`, false},
		{"country_visibility", `{"mode":"include","items":["US","RU"]}`, true},
		{"country_visibility", `{"mode":"include","items":[{}]}`, false},
		{"collection_contents", `{"1":[2,1],"2":[]}`, true},
		{"collection_contents", `{"01":[1]}`, false},
		{"collection_contents", `{"1":[2,2]}`, false},
		{"home_feed", `false`, true}, {"home_spotlight", `null`, false},
		{"home_watch_later", `"true"`, false},
		{"feed_page_size", `0`, false}, {"feed_page_size", `100`, true},
		{"home_collection_limit", `0`, true}, {"home_collection_limit", `31`, false},
	} {
		item := oc.Config{Scope: "frontend", Key: c.key, Title: "Frame", Value: json.RawMessage(c.value)}
		if err := item.ValidateForWrite(); (err == nil) != c.valid {
			t.Errorf("%s %s: %v", c.key, c.value, err)
		}
		if err := item.Validate(); err != nil {
			t.Errorf("legacy snapshot %s: %v", c.key, err)
		}
	}
}
