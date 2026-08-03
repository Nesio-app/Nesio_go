package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// EnsureQdrantCollection makes sure a target collection exists.
// It is safe to call on every startup.
func EnsureQdrantCollection(ctx context.Context, qdrantURL, collection string, vectorSize int) error {
	baseURL := strings.TrimRight(strings.TrimSpace(qdrantURL), "/")
	if strings.TrimSpace(collection) == "" {
		return fmt.Errorf("collection is required")
	}
	if vectorSize <= 0 {
		return fmt.Errorf("vector size must be positive")
	}

	candidateURLs := make([]string, 0, 2)
	if baseURL != "" {
		candidateURLs = append(candidateURLs, baseURL)
	} else {
		candidateURLs = append(candidateURLs, "http://qdrant:6333", "http://127.0.0.1:6333")
	}

	var lastErr error
	for _, url := range candidateURLs {
		if err := ensureCollectionAtURL(ctx, strings.TrimRight(url, "/"), collection, vectorSize); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("unable to ensure qdrant collection")
}

func ensureCollectionAtURL(ctx context.Context, baseURL, collection string, vectorSize int) error {

	client := &http.Client{Timeout: 8 * time.Second}
	collectionURL := fmt.Sprintf("%s/collections/%s", baseURL, collection)

	checkReq, err := http.NewRequestWithContext(ctx, http.MethodGet, collectionURL, nil)
	if err != nil {
		return err
	}
	checkResp, err := client.Do(checkReq)
	if err != nil {
		return err
	}
	checkResp.Body.Close()

	if checkResp.StatusCode == http.StatusOK {
		return nil
	}
	if checkResp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("qdrant collection check failed: %s", checkResp.Status)
	}

	payload := map[string]any{
		"vectors": map[string]any{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	createReq, err := http.NewRequestWithContext(ctx, http.MethodPut, collectionURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := client.Do(createReq)
	if err != nil {
		return err
	}
	defer createResp.Body.Close()

	if createResp.StatusCode >= 300 {
		return fmt.Errorf("qdrant collection create failed: %s", createResp.Status)
	}
	return nil
}
