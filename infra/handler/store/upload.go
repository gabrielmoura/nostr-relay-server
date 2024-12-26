package store

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/infra/metrics"
	"github.com/gabrielmoura/nostr-relay-server/infra/net"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/gabrielmoura/nostr-relay-server/pkg/magic"
	"github.com/goccy/go-json"
	"github.com/nbd-wtf/go-nostr"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func processAuth(w http.ResponseWriter, r *http.Request) {
	// TODO: retornar erros e tratar na função principal
	if config.Cfg.Ws.Auth {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" && !strings.HasPrefix(authHeader, "Nostr ") {
			http.Error(w, "Authorization header is required", http.StatusBadRequest)
			return
		}
		token := strings.TrimPrefix(authHeader, "Nostr ")

		decodedBytes, err := base64.StdEncoding.DecodeString(token)
		if err != nil {
			http.Error(w, "Failed to decode authorization", http.StatusBadRequest)
			log.Logger.Error("Decode error", zap.Error(err))
			return
		}
		var event nostr.Event
		err = json.Unmarshal(decodedBytes, &event)
		if err != nil {
			http.Error(w, "Failed to unmarshal authorization", http.StatusBadRequest)
			log.Logger.Error("Unmarshal error", zap.Error(err))
			return
		}
		if event.Kind != nostr.KindBlobs {
			http.Error(w, "Invalid event kind", http.StatusBadRequest)
			return
		}

		if ok, err := event.CheckSignature(); !ok || err != nil {
			http.Error(w, "Invalid signature", http.StatusBadRequest)
			return
		}
		// TODO: verificar se o pubkey é valido e autorizado a fazer upload
		// TODO: a tag x seraá o hash256 do arquivo
	}
}

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if !acceptMethods(r.Method, []string{http.MethodPost, http.MethodPut}) {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	startTime := time.Now()

	processAuth(w, r)

	file := r.Body
	defer file.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler o corpo da requisição", http.StatusBadRequest)
		log.Logger.Error("Read error", zap.Error(err))
		return
	}

	mgl, err := magic.Lookup(bodyBytes)
	if err != nil {
		//if !errors.Is(err, magic.ErrUnknown) {
		http.Error(w, "Failed to detect file type", http.StatusInternalServerError)
		log.Logger.Error("Magic lookup error", zap.Error(err))
		return
		//}
	}

	if !acceptMimeType(ternaryString(mgl.MIME, http.DetectContentType(bodyBytes)), config.Cfg.Store.AcceptedMimetypes) {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	// Use hash as the filename
	hasher := sha256.New()
	hasher.Write(bodyBytes)
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	filePath := filepath.Join(blobPath, hashString)
	size := int64(len(bodyBytes))
	mimeType := ternaryString(mgl.MIME, http.DetectContentType(bodyBytes))
	urlResponse := fmt.Sprintf("%s/%s", config.Cfg.Store.MediaPath, hashString)

	if _, err := os.Stat(filePath); err == nil {
		do, err := getFileExist(r.Context(), hashString)
		if err != nil {
			http.Error(w, "Failed to get file", http.StatusInternalServerError)
			log.Logger.Error("File get error", zap.Error(err))
			return
		}
		net.JsonResponse(w, http.StatusOK, do)
		return
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to create file on server", http.StatusInternalServerError)
		log.Logger.Error("File creation error", zap.Error(err))
		return
	}
	defer outFile.Close()

	_, err = outFile.Write(bodyBytes)
	if err != nil {
		http.Error(w, "Failed to write file content", http.StatusInternalServerError)
		log.Logger.Error("File write error", zap.Error(err))
		return
	}

	obj := &db2.Object{
		Hash:      hashString,
		MimeType:  mimeType,
		Size:      size,
		CreatedAt: time.Now(),
	}

	err = db.DbQueries.InsertObject(r.Context(), obj)
	if err != nil {
		http.Error(w, "Failed to save file metadata", http.StatusInternalServerError)
		//log.Println("DB save error:", err)
		log.Logger.Error("DB save error", zap.Error(err))
		return
	}

	response := &db2.ObjectResponse{
		Hash:      hashString,
		CreatedAt: obj.CreatedAt.Unix(),
		Url:       urlResponse,
		MimeType:  mimeType,
	}

	metrics.UploadCounter.Inc()
	net.JsonResponse(w, http.StatusOK, response)
	metrics.HttpDuration.WithLabelValues(r.URL.Path).Observe(time.Since(startTime).Seconds())
	return
}

func getFileExist(ctx context.Context, hash string) (*db2.ObjectResponse, error) {
	obj, err := db.DbQueries.GetObjectByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	urlResponse := fmt.Sprintf("%s/%s", config.Cfg.Store.MediaPath, hash)
	response := &db2.ObjectResponse{
		Hash:      hash,
		CreatedAt: obj.CreatedAt.Unix(),
		Url:       urlResponse,
		MimeType:  obj.MimeType,
	}

	return response, nil
}
