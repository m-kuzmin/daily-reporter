// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package github_graphql

import (
	"context"

	"github.com/Khan/genqlient/graphql"
)

// LoginResponse is returned by Login on success.
type LoginResponse struct {
	// The currently authenticated user.
	Viewer LoginViewerUser `json:"viewer"`
}

// GetViewer returns LoginResponse.Viewer, and is useful for accessing the field via an interface.
func (v *LoginResponse) GetViewer() LoginViewerUser { return v.Viewer }

// LoginViewerUser includes the requested fields of the GraphQL type User.
// The GraphQL type's documentation follows.
//
// A user is an individual's account on GitHub that owns repositories and can make new content.
type LoginViewerUser struct {
	// The username used to login.
	Login string `json:"login"`
}

// GetLogin returns LoginViewerUser.Login, and is useful for accessing the field via an interface.
func (v *LoginViewerUser) GetLogin() string { return v.Login }

// ViewerProjectsV2Response is returned by ViewerProjectsV2 on success.
type ViewerProjectsV2Response struct {
	// The currently authenticated user.
	Viewer ViewerProjectsV2ViewerUser `json:"viewer"`
}

// GetViewer returns ViewerProjectsV2Response.Viewer, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2Response) GetViewer() ViewerProjectsV2ViewerUser { return v.Viewer }

// ViewerProjectsV2ViewerUser includes the requested fields of the GraphQL type User.
// The GraphQL type's documentation follows.
//
// A user is an individual's account on GitHub that owns repositories and can make new content.
type ViewerProjectsV2ViewerUser struct {
	// A list of projects under the owner.
	ProjectsV2 ViewerProjectsV2ViewerUserProjectsV2ProjectV2Connection `json:"projectsV2"`
}

// GetProjectsV2 returns ViewerProjectsV2ViewerUser.ProjectsV2, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2ViewerUser) GetProjectsV2() ViewerProjectsV2ViewerUserProjectsV2ProjectV2Connection {
	return v.ProjectsV2
}

// ViewerProjectsV2ViewerUserProjectsV2ProjectV2Connection includes the requested fields of the GraphQL type ProjectV2Connection.
// The GraphQL type's documentation follows.
//
// The connection type for ProjectV2.
type ViewerProjectsV2ViewerUserProjectsV2ProjectV2Connection struct {
	// A list of edges.
	Edges []ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge `json:"edges"`
}

// GetEdges returns ViewerProjectsV2ViewerUserProjectsV2ProjectV2Connection.Edges, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2ViewerUserProjectsV2ProjectV2Connection) GetEdges() []ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge {
	return v.Edges
}

// ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge includes the requested fields of the GraphQL type ProjectV2Edge.
// The GraphQL type's documentation follows.
//
// An edge in a connection.
type ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge struct {
	// A cursor for use in pagination.
	Cursor string `json:"cursor"`
	// The item at the end of the edge.
	Node ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2 `json:"node"`
}

// GetCursor returns ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge.Cursor, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge) GetCursor() string {
	return v.Cursor
}

// GetNode returns ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge.Node, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2Edge) GetNode() ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2 {
	return v.Node
}

// ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2 includes the requested fields of the GraphQL type ProjectV2.
// The GraphQL type's documentation follows.
//
// New projects that manage issues, pull requests and drafts using tables and boards.
type ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2 struct {
	Id string `json:"id"`
	// The project's name.
	Title string `json:"title"`
}

// GetId returns ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2.Id, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2) GetId() string {
	return v.Id
}

// GetTitle returns ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2.Title, and is useful for accessing the field via an interface.
func (v *ViewerProjectsV2ViewerUserProjectsV2ProjectV2ConnectionEdgesProjectV2EdgeNodeProjectV2) GetTitle() string {
	return v.Title
}

// __ViewerProjectsV2Input is used internally by genqlient
type __ViewerProjectsV2Input struct {
	First int    `json:"first"`
	After string `json:"after"`
}

// GetFirst returns __ViewerProjectsV2Input.First, and is useful for accessing the field via an interface.
func (v *__ViewerProjectsV2Input) GetFirst() int { return v.First }

// GetAfter returns __ViewerProjectsV2Input.After, and is useful for accessing the field via an interface.
func (v *__ViewerProjectsV2Input) GetAfter() string { return v.After }

// The query or mutation executed by Login.
const Login_Operation = `
query Login {
	viewer {
		login
	}
}
`

func Login(
	ctx context.Context,
	client graphql.Client,
) (*LoginResponse, error) {
	req := &graphql.Request{
		OpName: "Login",
		Query:  Login_Operation,
	}
	var err error

	var data LoginResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}

// The query or mutation executed by ViewerProjectsV2.
const ViewerProjectsV2_Operation = `
query ViewerProjectsV2 ($first: Int! = 10, $after: String) {
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
}
`

func ViewerProjectsV2(
	ctx context.Context,
	client graphql.Client,
	first int,
	after string,
) (*ViewerProjectsV2Response, error) {
	req := &graphql.Request{
		OpName: "ViewerProjectsV2",
		Query:  ViewerProjectsV2_Operation,
		Variables: &__ViewerProjectsV2Input{
			First: first,
			After: after,
		},
	}
	var err error

	var data ViewerProjectsV2Response
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		ctx,
		req,
		resp,
	)

	return &data, err
}
