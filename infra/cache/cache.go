package cache

import (
	"context"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/dgraph-io/ristretto/v2"
)

// O cache agora é fortemente tipado para armazenar valores como []byte.
var cache *ristretto.Cache[string, []byte]
var initialized bool

// UserBanned define a estrutura para dados de banimento de usuário.
type UserBanned struct {
	Reason string `json:"r"`
	Banned bool   `json:"b"`
}

// GetUserBannedByKey é uma assinatura de função para buscar o status de banimento de um usuário.
type GetUserBannedByKey func(ctx context.Context, key string) (reason string, exists bool, err error)

// Init inicializa o cache global com a configuração para armazenar []byte.
func Init() error {
	var err error
	// A configuração agora especifica [string, []byte] para segurança de tipos.
	cache, err = ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: 1e7,      // Número de chaves para rastrear a frequência (10M).
		MaxCost:     10 << 20, // Custo máximo do cache (10MB).
		BufferItems: 64,       // Número de chaves por buffer Get.
	})
	if err == nil {
		initialized = true
	}
	return err
}

// Set armazena um valor []byte no cache. O custo é calculado como o comprimento do slice de bytes.
func Set(key string, value []byte) bool {
	if len(value) == 0 {
		return false
	}
	result := cache.Set(key, value, int64(len(value)))
	cache.Wait() // Aguarda o valor passar pelos buffers para garantir a escrita.
	return result
}

// Get recupera um valor []byte do cache.
func Get(key string) ([]byte, bool) {
	return cache.Get(key)
}

// Delete remove uma chave do cache.
func Delete(key string) {
	cache.Del(key)
	cache.Wait() // Garante que a exclusão seja processada.
}

// SetWithTTL armazena um valor []byte no cache com um tempo de vida (TTL) específico.
func SetWithTTL(key string, value []byte, ttl time.Duration) bool {
	if len(value) == 0 {
		return false
	}
	result := cache.SetWithTTL(key, value, int64(len(value)), ttl)
	cache.Wait() // Garante que a escrita seja processada.
	return result
}

// CheckSpam verifica se uma chave excedeu um limite de acesso, usando o cache para contagem.
func CheckSpam(key string, threshold int) (bool, error) {
	spamKey := key + "_spam"
	value, found := Get(spamKey)

	if !found {
		SetWithTTL(spamKey, []byte(strconv.Itoa(1)), time.Minute*5)
		return false, nil
	}

	count, err := strconv.Atoi(string(value))
	if err != nil {
		return false, err
	}

	if count >= threshold {
		return true, nil
	}

	newCount := strconv.Itoa(count + 1)
	SetWithTTL(spamKey, []byte(newCount), time.Minute*5)

	return false, nil
}

// GetBanned recupera o status de banimento de um usuário do cache.
func GetBanned(pubKey string) (reason string, banned bool, found bool) {
	rawJSON, foundInCache := Get(pubKey)
	if !foundInCache {
		return "", false, false
	}

	var userStatus UserBanned
	if err := json.Unmarshal(rawJSON, &userStatus); err != nil {
		return "", false, false
	}

	return userStatus.Reason, userStatus.Banned, true
}

// SetBanned armazena o status de banimento de um usuário no cache.
func SetBanned(pubKey string, val *UserBanned) error {
	rawJSON, err := json.Marshal(val)
	if err != nil {
		return err
	}
	SetWithTTL(pubKey, rawJSON, 5*time.Minute)
	return nil
}

// WrapGetBanned é um middleware que adiciona uma camada de cache à lógica de verificação de banimento.
func WrapGetBanned(internalLookup GetUserBannedByKey) GetUserBannedByKey {
	if initialized {
		return func(ctx context.Context, key string) (reason string, exists bool, err error) {
			bannedKey := key + "_banned"

			cachedReason, isBanned, foundInCache := GetBanned(bannedKey)

			if foundInCache {
				return cachedReason, isBanned, nil
			}

			reason, exists, err = internalLookup(ctx, key)
			if err != nil {
				return "", false, err
			}

			if err := SetBanned(bannedKey, &UserBanned{Reason: reason, Banned: exists}); err != nil {
				// Opcional: logar o erro do cache.
			}

			return reason, exists, nil
		}
	} else {
		return internalLookup
	}
}
