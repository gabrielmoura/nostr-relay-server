package store

var (
	blobPath   = "files"
	bufferSize = 2 << 9
)

func acceptMethods(method string, methods []string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

func acceptMimeType(mimeType string, acceptedMimetypes []string) bool {
	for _, v := range acceptedMimetypes {
		if mimeType == v {
			return true
		}
	}
	return false
}
