package github

import (
	"context"
	"net/http"

	genqlient "github.com/Khan/genqlient/graphql"
	graphql "github.com/m-kuzmin/daily-reporter/api/github"
)

type Client struct {
	client genqlient.Client
}

func NewClient(token string) Client {
	return Client{client: genqlient.NewClient("https://api.github.com/graphql",
		&http.Client{
			Transport: &authedTransport{token: token, wrapped: http.DefaultTransport}})}
}

type authedTransport struct {
	token   string
	wrapped http.RoundTripper
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.wrapped.RoundTrip(req)
}

func (c *Client) Login() (string, error) {
	_ = `# @genqlient
query Login {
  viewer {
    login
  }
}`
	if resp, err := graphql.Login(context.Background(), c.client); err != nil {
		return "", err
	} else {
		return resp.Viewer.Login, nil
	}
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
	_, err := graphql.ViewerProjectsV2(context.Background(), c.client, 10, "")
	if err != nil {
		panic("TODO:")
	}
	return make([]ProjectV2, 0)
}

type ProjectV2 struct {
	Cursor string
	Title  string
	Id     string
}
