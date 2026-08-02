package connector

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
