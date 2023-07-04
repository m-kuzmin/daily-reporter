package github

import (
	"net/http"

	genqlient "github.com/Khan/genqlient/graphql"
)

const githubGraphQLEndpoit = "https://api.github.com/graphql"

type Client struct {
	client genqlient.Client
}

func NewClient(token string) Client {
	return Client{client: genqlient.NewClient(githubGraphQLEndpoit,
		&http.Client{
			Transport: &authedTransport{token: token, wrapped: http.DefaultTransport},
		})}
}

type ProjectV2 struct {
	Cursor       ProjectCursor
	Title        string
	ID           ProjectID
	URL          string
	CreatorLogin string
	CreatorURL   string
	Number       int
}

type ProjectCursor string

type ProjectID string

// ProjectV2ItemsByStatus maps status names to a list of titles of items with that status.
type ProjectV2ItemsByStatus map[string][]string
