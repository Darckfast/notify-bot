package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"discordnotify/pkg/types"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	multilogger "github.com/Darckfast/multi_logger/pkg/multi_logger"
)

var logger = slog.New(multilogger.NewHandler(os.Stdout))

func GetTier(hook types.PatreonWebHook) string {
	var patreonTiers types.PatreonTiers

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
	ctx, wg := multilogger.SetupContext(&multilogger.SetupOps{
		Request:        r,
		BaselimeApiKey: os.Getenv("BASELIME_API_KEY"),
		AxiomApiKey:    os.Getenv("AXIOM_API_KEY"),
		ServiceName:    os.Getenv("VERCEL_GIT_REPO_SLUG"),
	})

	defer func() {
		wg.Wait()
		ctx.Done()
	}()

	logger.InfoContext(ctx, "Processing request")

	if r.Method == http.MethodHead {
		logger.InfoContext(ctx, "ping", "status", 200)
		return
	}

	if r.Method != http.MethodPost {
		logger.WarnContext(ctx, "Invalid method", "status", 200)
		return
	}

	if r.Header.Get("X-Patreon-Event") != "posts:publish" {
		logger.WarnContext(ctx, "Invalid event trigger", "status", 200)
		return
	}

	payloadSize, _ := strconv.Atoi(r.Header.Get("Content-Length"))

	if payloadSize == 0 || payloadSize > 1024*4 {
		logger.WarnContext(ctx, "Invalid request length", "status", 200)
		return
	}

	if r.Header.Get("User-Agent") != "Patreon HTTP Robot" {
		logger.WarnContext(ctx, "Invalid user agent", "status", 200)
		return
	}

	apiKey := r.URL.Query().Get("ak")

	if apiKey != os.Getenv("API_KEY") {
		logger.WarnContext(ctx, "Invalid api key", "status", 200)
		return
	}

	defer r.Body.Close()
	rawBody, _ := io.ReadAll(r.Body)
	patreonSig := r.Header.Get("X-Patreon-Signature")

	if !ValidatePayloadSignature(patreonSig, rawBody) {
		logger.WarnContext(ctx, "Invalid signature", "status", 200)
		return
	}

	var patreonHook types.PatreonWebHook

	if err := json.Unmarshal(rawBody, &patreonHook); err != nil {
		logger.ErrorContext(ctx, "Error decoding request payload", "error", err.Error())
		return
	}

	logger.InfoContext(ctx, "Processing request",
		"post", patreonHook.Data.ID,
		"publishedAt", patreonHook.Data.Attributes.PublishedAt,
	)

	var discPayload types.DiscordWebHook

	postUrl := "https://www.patreon.com/posts/" + patreonHook.Data.ID

	discEmbed := types.DiscordEmbed{
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
		patreonHook.Data.Attributes.PublishedAt.Format("02/01/2006 3:04 PM")

	discPayload.Content = os.Getenv("ALERT_MESSAGE")
	discPayload.Content += GetTier(patreonHook)
	discPayload.Content += "\n" + postUrl
	discPayload.Embeds = []types.DiscordEmbed{discEmbed}

	body, _ := json.Marshal(discPayload)

	discUrl := os.Getenv("DISCORD_WEBHOOK")
	req, _ := http.NewRequest("POST", discUrl, bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 3,
	}

	res, err := client.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "Error making request to discord", "error", err.Error())
		return
	}

	if res.StatusCode > 299 {
		w.WriteHeader(400)
		logger.ErrorContext(ctx, "Error firing alert to discord", "status", res.StatusCode)
		return
	}

	logger.InfoContext(ctx, "request completed", "status", 200, "postId", patreonHook.Data.ID)
}
