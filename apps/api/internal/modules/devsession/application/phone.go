package application

import (
	"errors"
	"strings"
)

var ErrInvalidPhone = errors.New("invalid synthetic mobile identifier")

// NormalizeSyntheticPhone accepts an Iranian 09xxxxxxxxx-shaped identifier.
// Local testing only: possession/ownership of the number is NEVER verified.
func NormalizeSyntheticPhone(input string) (string, error) {
	input = strings.TrimSpace(input)
	var digits strings.Builder
	for _, r := range input {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r >= '۰' && r <= '۹':
			digits.WriteRune('0' + r - '۰')
		case r >= '٠' && r <= '٩':
			digits.WriteRune('0' + r - '٠')
		default:
			return "", ErrInvalidPhone
		}
	}
	out := digits.String()
	if len(out) != 11 || !strings.HasPrefix(out, "09") {
		return "", ErrInvalidPhone
	}
	return out, nil
}
