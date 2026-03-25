package http

import (
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strings"

	adminui "github.com/gabrielmoura/nostr-relay-server/infra/dash"
	"github.com/gofiber/fiber/v2"
)

const adminUIBasePath = "/panel"

func AdminUIBasePath() string {
	return adminUIBasePath
}

func AdminUIDistDir() string {
	return filepath.Join("infra", "dash", "dist")
}

func adminUIFS() (fs.FS, error) {
	return adminui.DistFS()
}

func readEmbeddedAdminAsset(name string) ([]byte, error) {
	uiFS, err := adminUIFS()
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(uiFS, name)
}

func AdminUIIndex() fiber.Handler {
	return func(c *fiber.Ctx) error {
		content, err := readEmbeddedAdminAsset("index.html")
		if err != nil {
			indexPath := filepath.Join(AdminUIDistDir(), "index.html")
			if _, statErr := os.Stat(indexPath); statErr != nil {
				return c.Status(fiber.StatusServiceUnavailable).SendString("Admin UI dist not found. Build infra/dash first.")
			}
			return c.SendFile(indexPath)
		}
		c.Type("html", "utf-8")
		return c.Send(content)
	}
}

func AdminUIAsset() fiber.Handler {
	return func(c *fiber.Ctx) error {
		assetPath := strings.TrimPrefix(c.Path(), adminUIBasePath+"/")
		if assetPath == "" || assetPath == c.Path() {
			return fiber.ErrNotFound
		}

		content, err := readEmbeddedAdminAsset(assetPath)
		if err != nil {
			return fiber.ErrNotFound
		}

		if extension := filepath.Ext(assetPath); extension != "" {
			if contentType := mime.TypeByExtension(extension); contentType != "" {
				c.Set(fiber.HeaderContentType, contentType)
			}
		}

		return c.Send(content)
	}
}

func AdminUISPAFallback() fiber.Handler {
	return func(c *fiber.Ctx) error {
		trimmed := strings.TrimPrefix(c.Path(), adminUIBasePath)
		if strings.Contains(trimmed, ".") {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return AdminUIIndex()(c)
	}
}
