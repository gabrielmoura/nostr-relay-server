package http

import (
	"io/fs"
	"mime"
	"path"
	"strings"

	publicui "github.com/gabrielmoura/nostr-relay-server/infra/dash"
	"github.com/gofiber/fiber/v2"
)

func publicUIFS() (fs.FS, error) {
	return publicui.PublicDistFS()
}

// PublicUIIndex serves the public-only Vite entrypoint. Its bundle is distinct
// from the admin dashboard and therefore cannot expose admin routes or GraphQL
// client code on the relay listener.
func PublicUIIndex() fiber.Handler {
	return func(c *fiber.Ctx) error {
		uiFS, err := publicUIFS()
		if err != nil {
			return fiber.ErrNotFound
		}
		content, err := fs.ReadFile(uiFS, "public.html")
		if err != nil {
			return fiber.ErrNotFound
		}
		c.Type("html", "utf-8")
		return c.Send(content)
	}
}

// PublicUIAsset serves only assets emitted by the public Vite build.
func PublicUIAsset() fiber.Handler {
	return func(c *fiber.Ctx) error {
		assetPath := strings.TrimPrefix(c.Path(), "/assets/")
		if assetPath == "" || assetPath == c.Path() || strings.Contains(assetPath, "..") {
			return fiber.ErrNotFound
		}
		uiFS, err := publicUIFS()
		if err != nil {
			return fiber.ErrNotFound
		}
		content, err := fs.ReadFile(uiFS, path.Join("assets", assetPath))
		if err != nil {
			return fiber.ErrNotFound
		}
		if contentType := mime.TypeByExtension(path.Ext(assetPath)); contentType != "" {
			c.Set(fiber.HeaderContentType, contentType)
		}
		return c.Send(content)
	}
}
