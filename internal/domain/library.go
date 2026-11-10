package domain

import "errors"

var ErrInvalidLibraryData = errors.New("invalid library data")

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
