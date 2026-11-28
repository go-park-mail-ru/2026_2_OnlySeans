package domain

import "errors"

var ErrInvalidLibraryData = errors.New("invalid library data")

const LibraryAddKey = "frontend.library_add_enabled"

var ErrLibraryAddDisabled = errors.New("Добавление фильмов в личные списки выключено.")

type List string

const (
	Favorite   List = "favorite"
	WatchLater List = "watchlater"
	Watched    List = "watched"
)

type Library struct {
	Favorite   []int64 `json:"favorite"`
	WatchLater []int64 `json:"watchlater"`
	Watched    []int64 `json:"watched"`
}

type PreferenceKey string

const (
	PreferenceSidebarCollapsed PreferenceKey = "frame:sidebar-collapsed"
	PreferenceCatalogView      PreferenceKey = "frame:catalog-view"
)

type PreferenceValue string

const (
	PreferenceEnabled  PreferenceValue = "true"
	PreferenceDisabled PreferenceValue = "false"
	PreferenceGrid     PreferenceValue = `"grid"`
	PreferenceList     PreferenceValue = `"list"`
)
