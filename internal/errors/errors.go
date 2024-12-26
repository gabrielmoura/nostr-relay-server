package errors

import "errors"

var (
	ErrInvalidCanonicalURL = errors.New("invalid canonical URL")
	ErrInvalidURL          = errors.New("invalid URL")

	ErrorAuthHeaderRequired     = errors.New("Authorization header is required")
	ErrorDecodeAuthorization    = errors.New("Failed to decode authorization")
	ErrorUnmarshalAuthorization = errors.New("Failed to unmarshal authorization")
	ErrorInvalidEventKind       = errors.New("Invalid event kind")
	ErrorInvalidSignature       = errors.New("Invalid signature")
)
