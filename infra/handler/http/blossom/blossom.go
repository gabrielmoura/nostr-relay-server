package blossom

import (
	store "github.com/gabrielmoura/nostr-relay-server/infra/handler/store/blossom"
	"github.com/gofiber/fiber/v2"
)

func UploadHandler(c *fiber.Ctx) error {
	return store.UploadHandler(c)
}

func BlobHandler(c *fiber.Ctx) error {
	return store.BlobHandler(c)
}

func ReportHandler(c *fiber.Ctx) error {
	return store.ReportHandler(c)
}

func MirrorHandler(c *fiber.Ctx) error {
	return store.MirrorHandler(c)
}

func MediaHandler(c *fiber.Ctx) error {
	return store.MediaHandler(c)
}

func ListHandler(c *fiber.Ctx) error {
	return store.ListHandler(c)
}
