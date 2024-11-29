package patreon

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	multilogger "github.com/Darckfast/multi_logger/pkg/multi_logger"
)

var logger = slog.New(multilogger.NewHandler(os.Stdout))

func ValidatePayloadSignature(signature string, payload []byte) bool {
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	secret := os.Getenv("PATREON_WEBHOOK_SECRET")
	mac := hmac.New(md5.New, []byte(secret))

	mac.Write(payload)

	return hmac.Equal(sig, mac.Sum(nil))
}

func ValidateAndDecodePayload(ctx context.Context, r *http.Request) []byte {
	if r.Method == http.MethodHead {
		logger.InfoContext(ctx, "ping", "status", 200)
		return nil
	}

	if r.Method != http.MethodPost {
		logger.WarnContext(ctx, "Invalid method", "status", 200)
		return nil
	}

	if r.Header.Get("X-Patreon-Event") != "posts:publish" {
		logger.WarnContext(ctx, "Invalid event trigger", "status", 200)
		return nil
	}

	payloadSize, _ := strconv.Atoi(r.Header.Get("Content-Length"))

	if payloadSize == 0 || payloadSize > 1024*4 {
		logger.WarnContext(ctx, "Invalid request length", "status", 200)
		return nil
	}

	if r.Header.Get("User-Agent") != "Patreon HTTP Robot" {
		logger.WarnContext(ctx, "Invalid user agent", "status", 200)
		return nil
	}

	apiKey := r.URL.Query().Get("ak")

	if apiKey != os.Getenv("API_KEY") {
		logger.WarnContext(ctx, "Invalid api key", "status", 200)
		return nil
	}

	defer r.Body.Close()
	rawBody, _ := io.ReadAll(r.Body)
	patreonSig := r.Header.Get("X-Patreon-Signature")

	if !ValidatePayloadSignature(patreonSig, rawBody) {
		logger.WarnContext(ctx, "Invalid signature", "status", 200)
		return nil
	}

	return rawBody
}
