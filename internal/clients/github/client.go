package github

import (
	"context"
	"fmt"
	"net/http"

	genqlient "github.com/Khan/genqlient/graphql"
	graphql "github.com/m-kuzmin/daily-reporter/api/github"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
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

func (c Client) ListViewerProjects(first option.Option[int], after option.Option[ProjectCursor]) ([]ProjectV2, error) {
	_ = `# @genqlient
query ViewerProjectsV2($first: Int!, $after: String) {
  viewer {
    projectsV2(first: $first, after: $after) {
      edges {
        cursor
        node {
          id
          title
          number
          url
          creator {
            login
            url
          }
        }
      }
    }
  }
}`

	//nolint:gomnd
	graphql, err := graphql.ViewerProjectsV2(context.Background(), c.client, first.UnwrapOr(10),
		string(after.UnwrapOr("")))
	if err != nil {
		return []ProjectV2{}, fmt.Errorf("while requesting user's projects over GitHub GraphQL: %w", err)
	}

	projects := make([]ProjectV2, len(graphql.Viewer.ProjectsV2.Edges))

	for i, project := range graphql.Viewer.ProjectsV2.Edges {
		projects[i] = ProjectV2{
			Cursor:       ProjectCursor(project.Cursor),
			Title:        project.Node.Title,
			ID:           project.Node.Id,
			URL:          project.Node.Url,
			CreatorLogin: project.Node.Creator.GetLogin(),
			CreatorURL:   project.Node.Creator.GetUrl(),
			Number:       project.Node.Number,
		}
	}

	return projects, nil
}

type ProjectV2 struct {
	Cursor       ProjectCursor
	Title        string
	ID           string
	URL          string
	CreatorLogin string
	CreatorURL   string
	Number       int
}

type ProjectCursor string
