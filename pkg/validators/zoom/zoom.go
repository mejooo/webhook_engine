package zoom

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

const EnvToken = "ZOOM_WEBHOOK_SECRET_TOKEN"

type Config struct {
	Secret             string
	TolerateClockSkewS int
	LegacyV0Fallback   bool
}

func LoadSecretForCRC(fallback string) (string, error) {
	if v := os.Getenv(EnvToken); v != "" { return v, nil }
	if fallback != "" { return fallback, nil }
	return "", fmt.Errorf("%s not set", EnvToken)
}

func EncryptPlainToken(secret, plainToken string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(plainToken))
	return hex.EncodeToString(mac.Sum(nil))
}
