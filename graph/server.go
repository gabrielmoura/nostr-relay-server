package graph

import (
	"embed"
	"io/fs"
	"net/http"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

//go:embed schema_*.graphqls
var schemaFS embed.FS

func HTTPHandler() http.Handler {
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	return server
}

func PlaygroundHandler(endpoint string) http.Handler {
	return playground.Handler("Nostr Relay Admin GraphQL", endpoint)
}

func SchemaSDL() (string, error) {
	entries, err := fs.ReadDir(schemaFS, ".")
	if err != nil {
		return "", err
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".graphqls") {
			continue
		}
		paths = append(paths, entry.Name())
	}
	sort.Strings(paths)

	parts := make([]string, 0, len(paths))
	for _, path := range paths {
		contents, readErr := schemaFS.ReadFile(path)
		if readErr != nil {
			return "", readErr
		}
		parts = append(parts, string(contents))
	}

	return strings.Join(parts, "\n\n"), nil
}
