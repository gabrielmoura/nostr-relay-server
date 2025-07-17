package store

import (
	"github.com/gabrielmoura/nostr-relay-server/config"
	db2 "github.com/gabrielmoura/nostr-relay-server/infra/db"
	"github.com/gabrielmoura/nostr-relay-server/infra/log"
	"github.com/gabrielmoura/nostr-relay-server/internal/db"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"strconv"
	"time"
)

func ListHandler(c *fiber.Ctx) error {
	tags, pubKey, err := processAuth(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}
	tag := tags.GetFirst([]string{"t"}).Value()
	expiration := tags.GetFirst([]string{"expiration"}).Value()

	if expiration != "" {
		expirationTime, err := strconv.ParseInt(expiration, 10, 64)
		if err != nil {
			log.Logger.Error("Invalid expiration format", zap.Error(err))
			return c.Status(fiber.StatusBadRequest).SendString("Invalid expiration format")
		}
		if expirationTime < time.Now().Unix() {
			log.Logger.Warn("Expired request", zap.String("expiration", expiration), zap.String("remote_ip", c.IP()))
			return c.Status(fiber.StatusForbidden).SendString("Request expired")
		}
	}

	if tag != "list" {
		return c.Status(fiber.StatusBadRequest).SendString("Hashkey is required")
	}

	pub_key := c.Params("id")
	if pub_key == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Pubkey is required")
	}
	mediaList, err := db.DbQueries.GetAllObjectByKey(c.Context(), pubKey, 100)
	if err != nil {
		log.Logger.Error("Failed to retrieve media list", zap.Error(err), zap.String("pubkey", pubKey))
	}
	var response []MediaResponse
	for _, media := range mediaList {
		response = append(response, MediaResponse{
			Status:   "success",
			Message:  "",
			URL:      config.Cfg.Store.MediaPath + "/" + media.Hash,
			SHA256:   media.Hash,
			Size:     media.Size,
			Type:     media.MimeType,
			Uploaded: media.CreatedAt.Unix(),
			Blurhash: getParam(media, "blurhash"),
			Dim:      getParam(media, "dim"),
			NIP94:    getAllParams(media),
		})
	}
	if len(response) == 0 {
		c.Type(fiber.MIMEApplicationJSONCharsetUTF8)
		return c.Send([]byte("[]"))
	}
	return c.JSON(response)
}
func getParam(obj db2.Object, param string) string {
	if obj.Tags == nil || len(obj.Tags) == 0 {
		return ""
	}
	var tags map[string]string
	if err := json.Unmarshal(obj.Tags, &tags); err != nil {
		log.Logger.Error("Failed to unmarshal tags", zap.Error(err))
		return ""
	}
	if value, exists := tags[param]; exists {
		return value
	}
	return ""
}
func getAllParams(obj db2.Object) map[string]string {
	if obj.Tags == nil || len(obj.Tags) == 0 {
		return nil
	}
	var tags map[string]string
	if err := json.Unmarshal(obj.Tags, &tags); err != nil {
		log.Logger.Error("Failed to unmarshal tags", zap.Error(err))
		return nil
	}
	return tags
}

type MediaResponse struct {
	Status         string            `json:"status"`
	Message        string            `json:"message"`
	URL            string            `json:"url"`
	SHA256         string            `json:"sha256"`
	Size           int64             `json:"size"`
	Type           string            `json:"type"`
	Uploaded       int64             `json:"uploaded"`
	Blurhash       string            `json:"blurhash"`
	Dim            string            `json:"dim"`
	PaymentRequest string            `json:"payment_request"`
	Visibility     int               `json:"visibility"`
	NIP94          map[string]string `json:"nip94,omitempty"`
}
