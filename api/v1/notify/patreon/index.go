package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"discordnotify/pkg/patreon"
	"discordnotify/pkg/types"

	multilogger "github.com/Darckfast/multi_logger/pkg/multi_logger"
)

var logger = slog.New(multilogger.NewHandler(os.Stdout))

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx, wg := multilogger.SetupContext(&multilogger.SetupOps{
		Request:        r,
		BaselimeApiKey: os.Getenv("BASELIME_API_KEY"),
		AxiomApiKey:    os.Getenv("AXIOM_API_KEY"),
        BetterStackApiKey: os.Getenv("BETTERSTACK_API_KEY"),
		ServiceName:    os.Getenv("VERCEL_GIT_REPO_SLUG"),
	})

	defer func() {
		wg.Wait()
		ctx.Done()
	}()

	logger.InfoContext(ctx, "processing request")

	rawBody := patreon.ValidateAndDecodePayload(ctx, r)

	if rawBody == nil {
		return
	}

	var patreonHook types.PatreonWebHook
	if err := json.Unmarshal(rawBody, &patreonHook); err != nil {
		logger.ErrorContext(ctx, "error decoding request payload", "error", err.Error())
		return
	}

	logger.InfoContext(ctx, "creating webhook payload",
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
	discPayload.Content += patreon.MembersListInMD(patreonHook)
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
		logger.ErrorContext(ctx, "error making request to discord", "error", err.Error())
		return
	}

	if res.StatusCode > 299 {
		w.WriteHeader(400)
		logger.ErrorContext(ctx, "error firing alert to discord", "status", res.StatusCode)
		return
	}

	logger.InfoContext(ctx, "request completed", "status", 200, "postId", patreonHook.Data.ID)
}
