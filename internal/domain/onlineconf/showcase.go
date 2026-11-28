package onlineconf

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
)

type ShowcaseError string

func (e ShowcaseError) Error() string { return string(e) }

var countryCode = regexp.MustCompile("^[A-Z]{2}$")

func validFilmID(x any) bool {
	n, ok := x.(float64)
	return ok && n >= 1 && n <= float64(MaxSafeID) && math.Trunc(n) == n
}

func uniqueValues(value any, valid func(any) bool) bool {
	items, ok := value.([]any)
	if !ok {
		return false
	}
	seen := make(map[any]bool, len(items))
	for _, item := range items {
		if !valid(item) || seen[item] {
			return false
		}
		seen[item] = true
	}
	return true
}

func validateShowcase(c Config) error {
	if c.Scope != "frontend" {
		return nil
	}
	var v any
	switch c.Key {
	case "collection_visibility", "country_visibility", "collection_contents",
		"home_spotlight", "home_watch_later", "home_feed", "home_collection_limit", "feed_page_size":
		if err := json.Unmarshal(c.Value, &v); err != nil {
			return ErrInvalidJSON
		}
	default:
		return nil
	}
	switch c.Key {
	case "collection_visibility", "country_visibility":
		obj, _ := v.(map[string]any)
		mode, _ := obj["mode"].(string)
		valid := validFilmID
		if c.Key == "country_visibility" {
			valid = func(v any) bool { code, ok := v.(string); return ok && countryCode.MatchString(code) }
		}
		if len(obj) != 2 || (mode != "include" && mode != "exclude") || !uniqueValues(obj["items"], valid) {
			return ShowcaseError("Укажите режим include или exclude и список items без повторов.")
		}
	case "collection_contents":
		groups, ok := v.(map[string]any)
		if !ok {
			return ShowcaseError("Ожидается объект с составом подборок.")
		}
		for key, ids := range groups {
			id, err := strconv.ParseInt(key, 10, 64)
			if err != nil || id < 1 || id > MaxSafeID || strconv.FormatInt(id, 10) != key || !uniqueValues(ids, validFilmID) {
				return ShowcaseError("Для каждого ID подборки задайте список ID фильмов без повторов.")
			}
		}
	case "home_feed", "home_spotlight", "home_watch_later":
		if _, ok := v.(bool); !ok {
			return ShowcaseError("Ожидается значение true или false.")
		}
	default:
		min, max := 0, 30
		if c.Key == "feed_page_size" {
			min, max = 1, 100
		}
		n, ok := v.(float64)
		if !ok || math.Trunc(n) != n || n < float64(min) || n > float64(max) {
			return ShowcaseError(fmt.Sprintf("Введите целое число от %d до %d.", min, max))
		}
	}
	return nil
}
