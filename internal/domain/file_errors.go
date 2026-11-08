package domain

import "errors"

var ErrFileNotFound = errors.New("file not found")

// A failed commit can have an unknown outcome. Keep its S3 object until reconciled.
var ErrAvatarCommitUnknown = errors.New("avatar transaction commit outcome is unknown")
