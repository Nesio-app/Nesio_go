package connector

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Nesio-app/Nesio_go/internal/storage"
	"github.com/google/uuid"
)

type EncryptedPayload struct {
	Nonce  string `json:"nonce"`
	Cipher string `json:"cipher"`
}

func getConnectorKey() ([]byte, error) {
	key := os.Getenv("CONNECTOR_SECRET")
	if key == "" {
		return nil, fmt.Errorf("CONNECTOR_SECRET not set")
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("CONNECTOR_SECRET must be 32 bytes")
	}
	return []byte(key), nil
}

func EncryptCredentials(credentials map[string]any) (json.RawMessage, error) {
	if credentials == nil {
		return json.RawMessage(`{}`), nil
	}
	key, err := getConnectorKey()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, data, nil)
	payload := EncryptedPayload{
		Nonce:  base64.StdEncoding.EncodeToString(nonce),
		Cipher: base64.StdEncoding.EncodeToString(ciphertext),
	}
	return json.Marshal(payload)
}

func DecryptCredentials(raw json.RawMessage) (map[string]any, error) {
	var payload EncryptedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if payload.Nonce == "" || payload.Cipher == "" {
		// Backward compatibility for legacy plaintext jsonb records.
		var credentials map[string]any
		if err := json.Unmarshal(raw, &credentials); err != nil {
			return nil, err
		}
		return credentials, nil
	}
	key, err := getConnectorKey()
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Cipher)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	var credentials map[string]any
	if err := json.Unmarshal(plaintext, &credentials); err != nil {
		return nil, err
	}
	return credentials, nil
}

func MigrateLegacyCredentials(store *storage.Store) error {
	rows, err := store.DB.Queryx("SELECT id, credentials FROM connectors")
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var raw json.RawMessage
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}

		var payload EncryptedPayload
		if err := json.Unmarshal(raw, &payload); err == nil && payload.Nonce != "" && payload.Cipher != "" {
			continue
		}

		var credentials map[string]any
		if err := json.Unmarshal(raw, &credentials); err != nil {
			continue
		}
		encrypted, err := EncryptCredentials(credentials)
		if err != nil {
			return err
		}
		if _, err := store.DB.Exec("UPDATE connectors SET credentials = $2 WHERE id = $1", id, encrypted); err != nil {
			return err
		}
	}

	return rows.Err()
}
