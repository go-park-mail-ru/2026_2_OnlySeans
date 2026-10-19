package reviews

import (
	"fmt"
	"unicode/utf8"
)

const (
	minTitleLen   = 3
	maxTitleLen   = 255
	minContentLen = 50
	maxContentLen = 20000
)

func ValidateInput(input ReviewInput) error {
	titleLen := utf8.RuneCountInString(input.Title)
	if titleLen < minTitleLen {
		return fmt.Errorf("%w: too short, minimum %d characters", ErrInvalidTitle, minTitleLen)
	}
	if titleLen > maxTitleLen {
		return fmt.Errorf("%w: too long, maximum %d characters", ErrInvalidTitle, maxTitleLen)
	}

	contentLen := utf8.RuneCountInString(input.Content)
	if contentLen < minContentLen {
		return fmt.Errorf("%w: too short, minimum %d characters", ErrInvalidContent, minContentLen)
	}
	if contentLen > maxContentLen {
		return fmt.Errorf("%w: too long, maximum %d characters", ErrInvalidContent, maxContentLen)
	}
	return nil
}
