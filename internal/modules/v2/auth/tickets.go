package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type ticketEntry struct {
	Token     string
	ExpiresAt time.Time
}

var ticketStore sync.Map

func init() {
	// Goroutine de nettoyage pour supprimer les tickets expirés toutes les minutes
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			now := time.Now()
			ticketStore.Range(func(key, value interface{}) bool {
				entry := value.(ticketEntry)
				if now.After(entry.ExpiresAt) {
					ticketStore.Delete(key)
				}
				return true
			})
		}
	}()
}

// GenerateTicket crée un ticket court (valable 30 secondes) lié à un token JWT.
// Une erreur est retournée si l'aléa système est indisponible, afin de ne jamais
// émettre un ticket prévisible.
func GenerateTicket(tokenString string) (string, error) {
	bytes := make([]byte, 16) // 32 hex chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	ticket := hex.EncodeToString(bytes)

	ticketStore.Store(ticket, ticketEntry{
		Token:     tokenString,
		ExpiresAt: time.Now().Add(30 * time.Second),
	})

	return ticket, nil
}

// ConsumeTicket vérifie si un ticket existe et n'a pas expiré, puis le supprime (usage unique)
// Retourne le token JWT original et un booléen indiquant le succès.
func ConsumeTicket(ticket string) (string, bool) {
	val, ok := ticketStore.LoadAndDelete(ticket)
	if !ok {
		return "", false
	}

	entry := val.(ticketEntry)
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	return entry.Token, true
}
