package github

import (
	"context"
	"fmt"

	graphql "github.com/m-kuzmin/daily-reporter/api/github"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/pkg/errors"
)

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

func (c Client) ListViewerProjects(first uint, after option.Option[ProjectCursor]) ([]ProjectV2, error) {
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

	graphql, err := graphql.ViewerProjectsV2(context.Background(), c.client, int(first),
		string(after.UnwrapOr("")))
	if err != nil {
		return []ProjectV2{}, fmt.Errorf("while requesting user's projects over GitHub GraphQL: %w", err)
	}

	projects := make([]ProjectV2, len(graphql.Viewer.ProjectsV2.Edges))

	for i, project := range graphql.Viewer.ProjectsV2.Edges {
		projects[i] = ProjectV2{
			Cursor:       ProjectCursor(project.Cursor),
			Title:        project.Node.Title,
			ID:           ProjectID(project.Node.Id),
			URL:          project.Node.Url,
			CreatorLogin: project.Node.Creator.GetLogin(),
			CreatorURL:   project.Node.Creator.GetUrl(),
			Number:       project.Node.Number,
		}
	}

	return projects, nil
}

//nolint:funlen, cyclop // Yeah the filter is a bit complicated...
func (c Client) ListViewerProjectV2Items(
	ctx context.Context,
	projectID ProjectID,
	first uint,
	after option.Option[ProjectCursor],
) (ProjectV2ItemsByStatus, error) {
	_ = `# @genqlient
query GetProjectItems($id: ID!, $first: Int!, $after: String) {
  node(id: $id) {
    ... on ProjectV2 {
      items(first: $first, after: $after) {
        nodes {
          status: fieldValueByName(name: "Status") {
            ... on ProjectV2ItemFieldSingleSelectValue {
              name
            }
          }
          assignedTo: fieldValueByName(name: "Assignees") {
            ... on ProjectV2ItemFieldUserValue {
              users(first: 30) {
                nodes {
                  isViewer
                }
              }
            }
          }
          content {
            ... on DraftIssue {
              title
            }
            ... on Issue {
              title
            }
            ... on PullRequest {
              title
            }
          }
        }
        pageInfo {
          endCursor
          startCursor
          hasNextPage
        }
      }
    }
  }
}
`

	data, err := graphql.GetProjectItems(ctx, c.client, string(projectID), int(first),
		string(after.UnwrapOr("")))
	if err != nil {
		return ProjectV2ItemsByStatus{}, fmt.Errorf("while requesting user's projects over GitHub GraphQL: %w", err)
	}

	itemsByStatus := make(ProjectV2ItemsByStatus)
	//nolint:forcetypeassert // Schema says its only nil or a project.
	proj := data.Node.(*graphql.GetProjectItemsNodeProjectV2).Items.Nodes

	//nolint:lll // Has a lot of autogenerated types
	for _, node := range proj {
		if node.AssignedTo == nil || node.Content == nil || node.Status == nil {
			continue // Doesnt have all required fields
		}

		// The title of the issue
		var title string

		// Depending on the type of item in the board the type will be different but the title will be present.
		//nolint:forcetypeassert // Schema guarantees the types in this block
		switch node.Content.GetTypename() {
		case "DraftIssue":
			title = node.Content.(*graphql.GetProjectItemsNodeProjectV2ItemsProjectV2ItemConnectionNodesProjectV2ItemContentDraftIssue).Title

		case "Issue":
			title = node.Content.(*graphql.GetProjectItemsNodeProjectV2ItemsProjectV2ItemConnectionNodesProjectV2ItemContentIssue).Title

		case "PullRequest":
			title = node.Content.(*graphql.GetProjectItemsNodeProjectV2ItemsProjectV2ItemConnectionNodesProjectV2ItemContentPullRequest).Title
		default:
			continue // Something else which we dont care about.
		}

		// The name of the column in table view. It is stored as a single select value.
		statusGql, is := node.Status.(*graphql.GetProjectItemsNodeProjectV2ItemsProjectV2ItemConnectionNodesProjectV2ItemStatusProjectV2ItemFieldSingleSelectValue)
		if !is {
			continue // Dont know which column the item is in
		}

		status := statusGql.Name

		assignedTo, is := node.AssignedTo.(*graphql.GetProjectItemsNodeProjectV2ItemsProjectV2ItemConnectionNodesProjectV2ItemAssignedToProjectV2ItemFieldUserValue)
		if !is {
			continue // Noone is assigned or its a different type. We only need the ones with isViewer==true
		}

		for _, user := range assignedTo.Users.Nodes {
			if user.IsViewer {
				itemsByStatus[status] = append(itemsByStatus[status], title)

				break
			}
		}
	}

	return itemsByStatus, nil
}

func (c Client) ProjectV2ByID(ctx context.Context, id ProjectID) (ProjectV2, error) {
	_ = `# @genqlient
query ProjectV2ByID($id: ID!) {
  node(id: $id) {
    ... on ProjectV2 {
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
}`

	resp, err := graphql.ProjectV2ByID(ctx, c.client, string(id))
	if err != nil {
		return ProjectV2{}, errors.WithMessage(err, "while requesting ProjectV2 by ID")
	}

	project, is := resp.Node.(*graphql.ProjectV2ByIDNodeProjectV2)
	if !is {
		return ProjectV2{}, EmptyResponseError{
			Message: "while requesting projectv2 by ID the `node ... on TYPE` returned nil",
		}
	}

	return ProjectV2{
		Cursor:       "",
		Title:        project.Title,
		ID:           ProjectID(project.Id),
		URL:          project.Url,
		CreatorLogin: project.Creator.GetLogin(),
		CreatorURL:   project.GetCreator().GetUrl(),
		Number:       project.Number,
	}, nil
}
