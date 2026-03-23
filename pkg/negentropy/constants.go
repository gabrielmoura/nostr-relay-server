package negentropy

const (
	ProtocolVersion = "/negentropy/1.0.0"
	FrameSizeLimit  = 1024 * 1024 // 1MB
	IDSize          = 32

	// Tipos de Mensagem NIP-77
	MsgOpen  = "NEG-OPEN"
	MsgMsg   = "NEG-MSG"
	MsgErr   = "NEG-ERR"
	MsgClose = "NEG-CLOSE"
	MsgHave  = "NEG-HAVE"
	MsgNeed  = "NEG-NEED"
)
