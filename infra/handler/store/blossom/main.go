package blossom

import "slices"

var (
	blobPath   = "files"
	bufferSize = 2 << 9
)

func acceptMethods(method string, methods []string) bool {
	return slices.Contains(methods, method)
}

func acceptMimeType(mimeType string, acceptedMimetypes []string) bool {
	return slices.Contains(acceptedMimetypes, mimeType)
}
