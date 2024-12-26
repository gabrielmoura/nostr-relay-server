package store

import (
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ByteRange struct {
	Start  int64
	Length int64
}

func parseRange(rangeHeader string, fileSize int64) ([]ByteRange, error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("invalid range header")
	}

	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	ranges := strings.Split(rangeHeader, ",")
	var byteRanges []ByteRange

	for _, part := range ranges {
		parts := strings.Split(strings.TrimSpace(part), "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range format")
		}

		start, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid start range")
		}

		var end int64
		if parts[1] == "" {
			end = fileSize - 1
		} else {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid end range")
			}
		}

		if start > end || start < 0 || end >= fileSize {
			return nil, fmt.Errorf("range out of bounds")
		}

		byteRanges = append(byteRanges, ByteRange{
			Start:  start,
			Length: end - start + 1,
		})
	}

	return byteRanges, nil
}

func serveFile(w http.ResponseWriter, filePath string, ranges []ByteRange, mime string) {
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to retrieve file info", http.StatusInternalServerError)
		return
	}

	if len(ranges) == 0 {
		// Serve entire file
		w.Header().Set("Content-Type", ternaryString(mime, "application/octet-stream"))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, file)
		return
	}

	// Serve specified ranges
	for _, r := range ranges {
		sectionReader := io.NewSectionReader(file, r.Start, r.Length)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", r.Start, r.Start+r.Length-1, fileInfo.Size()))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", r.Length))
		w.Header().Set("Content-Type", ternaryString(mime, "application/octet-stream"))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.Copy(w, sectionReader)
		return // Serve only the first range if multiple are specified
	}
}

func BlobHandler(w http.ResponseWriter, r *http.Request) {
	if !acceptMethods(r.Method, []string{http.MethodHead, http.MethodGet}) {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/blob/")
	if id == "" {
		http.Error(w, "Invalid file ID", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(blobPath, id)
	if r.Method == http.MethodHead {
		if _, err := os.Stat(filePath); err != nil {
			w.WriteHeader(http.StatusNotFound)

		} else {
			w.WriteHeader(http.StatusOK)
		}
		w.Write([]byte(""))
		return
	}

	// Simulated object retrieval from a database.
	o, err := db.DbQueries.GetObjectByHash(r.Context(), id)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	if o.Blocked {
		http.Error(w, "File is blocked", http.StatusForbidden)
		return
	}
	if !o.ExpiresAt.IsZero() && time.Now().After(o.ExpiresAt) {
		go func() {
			if err := os.Remove(filePath); err != nil {
				log.Println("Failed to remove file", err)
			}

			if err := db.DbQueries.RemoveObject(r.Context(), id); err != nil {
				log.Println("Failed to remove object", err)
			}
		}()
		http.Error(w, "File has expired", http.StatusGone)
		return
	}
	if o.BlockedByReason != "" {
		http.Error(w, o.BlockedByReason, http.StatusUnavailableForLegalReasons)
		return
	}
	metrics.DownloadCounter.Inc()

	var ranges []ByteRange

	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		file, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			http.Error(w, "Unable to retrieve file info", http.StatusInternalServerError)
			return
		}

		ranges, err = parseRange(rangeHeader, fileInfo.Size())
		if err != nil {
			http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	serveFile(w, filePath, ranges, o.MimeType)
}

func ternaryString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
