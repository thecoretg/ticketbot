package connectwise

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func callbackIdEndpoint(callbackID int) string {
	return fmt.Sprintf("system/callbacks/%d", callbackID)
}

func (c *Client) PostCallback(typeArg *Callback) (*Callback, error) {
	return Post[Callback](c, "system/callbacks", typeArg)
}

func (c *Client) ListCallbacks(params map[string]string) ([]Callback, error) {
	return GetMany[Callback](c, "system/callbacks", params)
}

func (c *Client) GetCallback(callbackID int, params map[string]string) (*Callback, error) {
	return GetOne[Callback](c, callbackIdEndpoint(callbackID), params)
}

func (c *Client) PutCallback(callbackID int, typeArg *Callback) (*Callback, error) {
	return Put[Callback](c, callbackIdEndpoint(callbackID), typeArg)
}

func (c *Client) PatchCallback(callbackID int, patchOps []PatchOp) (*Callback, error) {
	return Patch[Callback](c, callbackIdEndpoint(callbackID), patchOps)
}

func (c *Client) DeleteCallback(callbackID int) error {
	return Delete(c, callbackIdEndpoint(callbackID))
}

func ValidateWebhook(r *http.Request) (bool, error) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("reading request body: %w", err)
	}
	defer r.Body.Close()

	var meta struct {
		Metadata struct {
			KeyURL string `json:"key_url"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return false, fmt.Errorf("unmarshaling request body: %w", err)
	}

	resp, err := http.Get(meta.Metadata.KeyURL)
	if err != nil {
		return false, fmt.Errorf("getting shared secret key from %s: %w", meta.Metadata.KeyURL, err)
	}
	defer resp.Body.Close()

	var keyResp struct {
		SigningKey string `json:"signing_key"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&keyResp); err != nil {
		return false, fmt.Errorf("decoding shared secret key response: %w", err)
	}

	sharedSecret := []byte(keyResp.SigningKey)
	hash := sha256.Sum256(sharedSecret)
	h := hmac.New(sha256.New, hash[:])
	h.Write(payload)
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	expectedSig := r.Header.Get("x-content-signature")

	return signature == expectedSig, nil
}
