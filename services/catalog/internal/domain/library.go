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
	Favorite   []FilmID
	WatchLater []FilmID
	Watched    []FilmID
}
