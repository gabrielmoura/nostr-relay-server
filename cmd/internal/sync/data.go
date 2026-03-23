package sync

// ConfSync define a configuração para uma operação de sincronização
type ConfSync struct {
	Remote    string // URL do Relay Remoto (wss://...)
	Pk        string // Chave Pública (hex ou npub) para filtrar
	Direction string // "up", "down" ou "both" (atualmente focado em bidirecional)
}
