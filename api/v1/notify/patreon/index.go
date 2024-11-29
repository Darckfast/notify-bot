package patreon

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

type MsgOpts struct {
	AuthorName    string
	AuthorUrl     string
	AuthorIconUrl string
	NotifyMessage string
	WebhookUrl    string
	ImageUrl      string
	ThumbnailUrl  string
	AtRoleId      string
}

var MessageOpts = MsgOpts{
	ImageUrl:      os.Getenv("BANNER_IMAGE_URL"),
	ThumbnailUrl:  os.Getenv("THUMBNAIL_IMAGE_URL"),
	AuthorName:    os.Getenv("PATREON_NAME"),
	AuthorUrl:     os.Getenv("PATREON_URL"),
	AuthorIconUrl: os.Getenv("PATREON_ICON_URL"),
	NotifyMessage: os.Getenv("MESSAGE"),
	WebhookUrl:    os.Getenv("DISCORD_WEBHOOK"),
	AtRoleId:      os.Getenv("@ROLE_ID"),
}

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx, wg := multilogger.SetupContext(&multilogger.SetupOps{
		Request:           r,
		BaselimeApiKey:    os.Getenv("BASELIME_API_KEY"),
		AxiomApiKey:       os.Getenv("AXIOM_API_KEY"),
		BetterStackApiKey: os.Getenv("BETTERSTACK_API_KEY"),
		ServiceName:       os.Getenv("VERCEL_GIT_REPO_SLUG"),
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
	discEmbed.Image.URL = MessageOpts.ImageUrl
	discEmbed.Thumbnail.URL = MessageOpts.ThumbnailUrl
	discEmbed.Provider.Name = "Patreon"
	discEmbed.Provider.URL = "https://patreon.com"
	discEmbed.Author.Name = MessageOpts.AuthorName
	discEmbed.Author.URL = MessageOpts.AuthorUrl
	discEmbed.Author.IconURL = MessageOpts.AuthorIconUrl
	discEmbed.Footer.Text = "Patreon • " +
		patreonHook.Data.Attributes.PublishedAt.Format("02/01/2006 3:04 PM")

	if MessageOpts.AtRoleId != "" {
		discPayload.Content += "<@&" + MessageOpts.AtRoleId + "> "
	}

	discPayload.Content += MessageOpts.NotifyMessage
	discPayload.Content += "\n" + postUrl
	discPayload.Embeds = []types.DiscordEmbed{discEmbed}

	body, _ := json.Marshal(discPayload)

	discUrl := MessageOpts.WebhookUrl
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
		logger.ErrorContext(ctx, "error firing alert to discord", "status", res.StatusCode)
		return
	}

	logger.InfoContext(ctx, "request completed", "status", 200, "postId", patreonHook.Data.ID)
}
