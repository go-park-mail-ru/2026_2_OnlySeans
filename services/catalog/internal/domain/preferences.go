package domain

import "errors"

var ErrInvalidPreferences = errors.New("invalid preferences")

type CatalogView string

const (
	CatalogViewGrid CatalogView = "grid"
	CatalogViewList CatalogView = "list"
)

type Preferences struct {
	SidebarCollapsed *bool
	CatalogView      *CatalogView
}

type PreferenceUpdate struct {
	SidebarCollapsed *bool
	CatalogView      *CatalogView
}

func (update PreferenceUpdate) Validate() error {
	if (update.SidebarCollapsed == nil) == (update.CatalogView == nil) {
		return ErrInvalidPreferences
	}

	if update.CatalogView != nil {
		switch *update.CatalogView {
		case CatalogViewGrid, CatalogViewList:
		default:
			return ErrInvalidPreferences
		}
	}

	return nil
}
