package github

import (
	"context"
	"fmt"
	"net/http"

	genqlient "github.com/Khan/genqlient/graphql"
	graphql "github.com/m-kuzmin/daily-reporter/api/github"
	"github.com/pkg/errors"
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

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform RoundTrip in authedTransport")
	}

	return resp, nil
}

func (c *Client) Login() (string, error) {
	_ = `# @genqlient
query Login {
  viewer {
    login
  }
}`

	resp, err := graphql.Login(context.Background(), c.client)
	if err != nil {
		return "", fmt.Errorf("while getting user's GitHub username (login): %w", err)
	}

	return resp.Viewer.Login, nil
}

func (c *Client) ListViewerProjects() []ProjectV2 {
	_ = `# @genqlient
query ViewerProjectsV2($first: Int! = 10, $after: String) {
  viewer {
	projectsV2(first: $first, after: $after) {
      edges {
		cursor
        node {
          id
          title
        }
      }
    }
  }
}`

	const (
		first = 10
		after = ""
	)

	_, err := graphql.ViewerProjectsV2(context.Background(), c.client, first, after)
	if err != nil {
		panic("TODO:")
	}

	return make([]ProjectV2, 0)
}

type ProjectV2 struct {
	Cursor string
	Title  string
	ID     string
}
