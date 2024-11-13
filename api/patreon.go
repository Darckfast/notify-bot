package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

var tempTiers = `[{"attributes":{"title": "Free"},"id": "24376423"},{"attributes":{"title":"tier-1"},"id":"24377500"}]`

func GetTier(hook PatreonWebHook) string {
	var patreonTiers PatreonTiers

	json.Unmarshal([]byte(tempTiers), &patreonTiers)

	tierString := "**"

	for _, postTier := range hook.Data.Attributes.Tiers {
		for _, tier := range patreonTiers {
			tierId, _ := strconv.Atoi(tier.Id)
			if tierId == postTier {
				tierString += " " + tier.Attributes.Title
			}
		}
	}

	return tierString + "**: "
}

// block ip
// validate payload
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}

    fmt.Println(r.RemoteAddr)
	// Loop over header names
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			fmt.Println(name, value)
		}
	}

	defer r.Body.Close()

	discUrl := os.Getenv("DISCORD_WEBHOOK")

	var patreonHook PatreonWebHook

	if err := json.NewDecoder(r.Body).Decode(&patreonHook); err != nil {
		// add log
		return
	}

	var discPayload DiscordWebHook

	discPayload.Content = os.Getenv("ALERT_MESSAGE")
	discPayload.Content += GetTier(patreonHook)
	discPayload.Content += "\nhttps://www.patreon.com/posts/" + patreonHook.Data.ID

	body, _ := json.Marshal(discPayload)

	req, _ := http.NewRequest("POST", discUrl, bytes.NewBuffer(body))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{
		Timeout: time.Second * 3,
	}

	res, _ := client.Do(req)
	defer res.Body.Close()

	resBody, _ := io.ReadAll(res.Body)
	w.Write(resBody)
}
