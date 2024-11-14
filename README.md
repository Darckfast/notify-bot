# Patreon -> Discord notify bot
![Pasted image 20241114175248](https://github.com/user-attachments/assets/8cde3136-0504-4cd1-b9e4-c49bd50eb278)

This middleware functions allows you to use the Patreon's webhooks to feed notification into discord, and it doesn't require write/read permissions to Patreon's API

## Setting up webhooks on Patreon
Unfortunaly at the moment Patreon does not let you create the webhook with the required triggers on their portal page, so you must do throught their API

### Create a Client on Patreon
Navigate to https://www.patreon.com/portal and go to "My Clients"

![Pasted image 20241114172208](https://github.com/user-attachments/assets/37289981-317d-4c6b-b289-4e3da7cb6185)

Click on "Create Client"

![Pasted image 20241114172254](https://github.com/user-attachments/assets/6cf6d3dd-9447-47a6-b15b-5ad2a517c92e)

Fill the form, the information on the form itself is not relevant nor matters for what we need to do

Click on your client and copy the "Client Secret", this will be used to authenticate our request, for you to create the proper webhook

**CAREFUL, THIS TOKEN SHOULD NOT BE POSTED ANYWHERE, AS IT ALLOW ANYONE TO REQUEST PATREON API IN YOUR BEHALF, AND IT CAN ACCESS SENTISIVE INFORMATION ABOUT YOU AND YOUR MEMBERS**

![Pasted image 20241114172942](https://github.com/user-attachments/assets/9af89b0f-46f6-49d6-b96e-8e7120fb1e21)

### Setting up the webhook on Patreon
Before creating the webhook, we first need the know the "campaign" id of your page, to do that, make this GET request, using the token you copy from the previous step 

**Request**
```http
GET /api/oauth2/v2/campaigns HTTP/1.1
User-Agent: insomnia/10.1.1
Authorization: Bearer creator_access_token
Host: www.patreon.com
```

**Response**
```json
{
	"data": [
		{
			"attributes": {},
			"id": "13117666",
			"type": "campaign"
		}
	],
	"meta": {
		"pagination": {
			"cursors": {
				"next": null
			},
			"total": 1
		}
	}
}
```

Before creating the webhook, the function need to have the ENV `API_KEY` set with any string, this will be included as query param when setting up the webhook on Patreon

To create the webhook, make the following request, the `uri` field must be a full URL to your function, including the query param `ak` which should be the same as the ENV `API_KEY`

**Request**
```http
POST /api/oauth2/v2/webhooks HTTP/1.1
Content-Type: application/json
Authorization: Bearer creator_access_token
Host: www.patreon.com
Content-Length: 311

{
  "data": {
    "type": "webhook",
    "attributes": {
      "triggers": ["posts:publish"],
      "uri": "https://your_url_to_this_function?ak=api_key_configured_on_this_function"
    },
    "relationships": {
      "campaign": {
        "data": {"type": "campaign", "id": "13117666"}
      }
    }
  }
}
```

**Response**
```json
{
	"data": {
		"attributes": {
			"last_attempted_at": null,
			"num_consecutive_times_failed": 0,
			"paused": false,
			"secret": "webhook_secret",
			"triggers": [
				"posts:publish"
			],
			"uri": "https://your_url_to_this_function?ak=api_key_configured_on_this_function"
		},
		"id": "756209",
		"type": "webhook"
	},
	"links": {
		"self": "https://www.patreon.com/api/oauth2/v2/webhooks/756209"
	}
}
```

From the response, you must set the ENV `PATREON_WEBHOOK_SECRET` with the `secret` value, this will be used to validate Patreon's webhook payload, and guarantee that Patreon is the source of the request


### Setting up Discord webhook
On "Integrations > Webhooks", create a new Webhook, set the channel, name, image and copy the webhook URL

![Pasted image 20241114174449](https://github.com/user-attachments/assets/cdf7a80d-2bb1-4501-b0f0-9eaa8d0da018)

The ENV `DISCORD_WEBHOOK` must be set with the URL copied above

### Customizing the message
The following ENVs must be set to customize the alert message and embed information

```bash
ALERT_MESSAGE="Check out new post on Patreon"

BANNER_IMAGE_URL=url_to_image

THUMBNAIL_IMAGE_URL=url_to_image

PATREON_NAME=Your patreon name

PATREON_URL=https://www.patreon.com/c/your_patreon

PATREON_ICON_URL=url_to_image
```
