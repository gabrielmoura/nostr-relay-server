package version

import "runtime"

// goVersion retorna a versão do toolchain Go usada para compilar o binário
// (ex: "go1.23.4"). Isso vem embutido no binário pelo próprio compilador,
// não depende de -ldflags, e não vaza nenhum path do ambiente de build.
func goVersion() string {
	return runtime.Version()
}
