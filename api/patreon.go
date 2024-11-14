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

type DiscordEmbed struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Color int    `json:"color"`
	Image struct {
		URL string `json:"url"`
	} `json:"image"`
	Thumbnail struct {
		URL string `json:"url"`
	} `json:"thumbnail"`
	Provider struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"provider"`
	Author struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		IconURL string `json:"icon_url"`
	} `json:"author"`
	Footer struct {
		Text string `json:"text"`
	} `json:"footer"`
}

type DiscordWebHook struct {
	Content string         `json:"content"`
	Embeds  []DiscordEmbed `json:"embeds"`
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

	postUrl := "https://www.patreon.com/posts/" + patreonHook.Data.ID

	discEmbed := DiscordEmbed{
		Title: patreonHook.Data.Attributes.Title,
		URL:   postUrl,
		Color: 16345172,
	}
	discEmbed.Image.URL = os.Getenv("BANNER_IMAGE_URL")
	discEmbed.Thumbnail.URL = os.Getenv("THUMBNAIL_IMAGE_URL")
	discEmbed.Provider.Name = "Patreon"
	discEmbed.Provider.URL = "https://patreon.com"
	discEmbed.Author.Name = os.Getenv("PATREON_NAME")
	discEmbed.Author.URL = os.Getenv("PATREON_URL")
	discEmbed.Author.IconURL = os.Getenv("PATREON_ICON_URL")
	discEmbed.Footer.Text = "Patreon • " +
		patreonHook.Data.Attributes.PublishedAt.Format("02/01/2022 3:04 PM")

	discPayload.Content = os.Getenv("ALERT_MESSAGE")
	discPayload.Content += GetTier(patreonHook)
	discPayload.Content += "\n" + postUrl
	discPayload.Embeds = []DiscordEmbed{discEmbed}

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
