package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

const (
	HeaderTimestamp  = "X-GB-Timestamp"
	HeaderNonce      = "X-GB-Nonce"
	HeaderSignature  = "X-GB-Signature"
	HeaderServerUUID = "X-GB-Server-UUID"
)

type SignedRequest struct {
	Timestamp string
	Nonce     string
	Signature string
	Body      []byte
}

func Sign(token, serverUUID string, body []byte) (SignedRequest, error) {
	if token == "" {
		return SignedRequest{}, fmt.Errorf("agent token is empty")
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce, err := randomNonce(16)
	if err != nil {
		return SignedRequest{}, err
	}
	payload := ts + "." + nonce + "." + string(body)
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write([]byte(payload))
	return SignedRequest{
		Timestamp: ts,
		Nonce:     nonce,
		Signature: hex.EncodeToString(mac.Sum(nil)),
		Body:      body,
	}, nil
}

func randomNonce(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
