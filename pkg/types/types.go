package types

import "time"

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
