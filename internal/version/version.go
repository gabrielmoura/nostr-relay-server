// Package version expõe metadados de build do NRServer.
//
// Os valores das variáveis abaixo são preenchidos em tempo de compilação
// via -ldflags -X (ver Makefile). Em builds locais com `go run` ou `go build`
// sem essas flags, eles mantêm os defaults declarados aqui.
package version

var (
	// Version é a tag/versão do binário (ex: v0.2.0, ou v0.2.0-3-gabc1234-dirty).
	Version = "dev"

	// Commit é o hash curto do commit git usado no build.
	Commit = "unknown"

	// BuildTime é o timestamp UTC do build, formato RFC3339.
	BuildTime = "unknown"
)

// Info agrupa os metadados de versão em um struct, útil para serializar
// (ex: em JSON, no endpoint NIP-11 do relay ou em `nrserver --version`).
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// Get retorna os metadados de versão atuais.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: goVersion(),
	}
}

// String formata a versão em uma linha curta, adequada para `--version`
// e para logs de startup.
func (i Info) String() string {
	return "nrserver " + i.Version + " (commit " + i.Commit + ", built " + i.BuildTime + ", " + i.GoVersion + ")"
}
