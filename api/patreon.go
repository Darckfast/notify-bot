package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PatreonWebHook struct {
	Data struct {
		Attributes struct {
			IsPaid      bool      `json:"is_paid"`
			IsPublic    bool      `json:"is_public"`
			PublishedAt time.Time `json:"published_at"`
			Tiers       []int     `json:"tiers"`
			Title       string    `json:"title"`
			URL         string    `json:"url"`
		} `json:"attributes"`
		ID string `json:"id"`
	} `json:"data"`
}

type DiscordWebHook struct {
	Content string `json:"content"`
}

type PatreonTiers []struct {
	Id         string `json:"id"`
	Attributes struct {
		Title string `json:"title"`
	} `json:"attributes"`
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func GetTier(hook PatreonWebHook) string {
	var patreonTiers PatreonTiers

	json.Unmarshal([]byte(os.Getenv("TIERS")), &patreonTiers)

	tierString := "**"

	for _, postTier := range hook.Data.Attributes.Tiers {
		for _, tier := range patreonTiers {
			tierId, _ := strconv.Atoi(tier.Id)
			if tierId == postTier {
				tierString += " " + tier.Attributes.Title
			}
		}
	}

	if tierString == "**" {
		return ""
	}

	return tierString + "**: "
}

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

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Warn("Invalid method")
		return
	}

	if r.Header.Get("X-Patreon-Event") != "posts:publish" {
		logger.Warn("Invalid event trigger")
		return
	}

	payloadSize, _ := strconv.Atoi(r.Header.Get("Content-Length"))

	if payloadSize == 0 || payloadSize > 1024*4 {
		logger.Warn("Invalid request length")
		return
	}

	if r.Header.Get("User-Agent") != "Patreon HTTP Robot" {
		logger.Warn("Invalid user agent")
		return
	}

	apiKey := r.URL.Query().Get("ak")

	if apiKey != os.Getenv("API_KEY") {
		logger.Warn("Invalid api key")
		return
	}

	defer r.Body.Close()
	rawBody, _ := io.ReadAll(r.Body)
	patreonSig := r.Header.Get("X-Patreon-Signature")

	if !ValidatePayloadSignature(patreonSig, rawBody) {
		logger.Warn("Invalid signature")
		return
	}

	var patreonHook PatreonWebHook

	if err := json.Unmarshal(rawBody, &patreonHook); err != nil {
		logger.Error("Error decoding request payload", "error", err.Error())
		return
	}

	logger.Info("Processing request",
		"post", patreonHook.Data.ID,
		"publishedAt", patreonHook.Data.Attributes.PublishedAt,
	)

	var discPayload DiscordWebHook

	discPayload.Content = os.Getenv("ALERT_MESSAGE")
	discPayload.Content += GetTier(patreonHook)
	discPayload.Content += "\nhttps://www.patreon.com/posts/" + patreonHook.Data.ID

	body, _ := json.Marshal(discPayload)

	discUrl := os.Getenv("DISCORD_WEBHOOK")
	req, _ := http.NewRequest("POST", discUrl, bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 3,
	}

	_, err := client.Do(req)
	if err != nil {
		logger.Error("Error making request to discord", "error", err.Error())
		return
	}

	logger.Info("Alert sent", "post", patreonHook.Data.ID)
}
