package domain

type PosterFile struct {
	ExternalURL *string
	Filename    string
	ContentType string
	Size        int64
}
