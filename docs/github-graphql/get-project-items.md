# Obtaining a list of project items

```graphql
{
  node(id: "PVT_kwHOBDyM384AP9Et") {
    ... on ProjectV2 {
      items(first: 60) {
        pageInfo {
          hasNextPage
        }
        edges {
          node {
            content {
              ... on DraftIssue {
                title
                id
                assignees(first: 50) {
                  pageInfo {
                    hasNextPage
                  }
                  edges {
                    node {
                      login
                    }
                  }
                }
              }
              ... on Issue {
                title
                id
                assignees(first: 50) {
                  pageInfo {
                    hasNextPage
                  }
                  edges {
                    node {
                      login
                    }
                  }
                }
              }
              ... on PullRequest {
                title
                id
                assignees(first: 50) {
                  pageInfo {
                    hasNextPage
                  }
                  edges {
                    node {
                      login
                    }
                  }
                }
              }
            }
            id
            type
          }
        }
      }
    }
  }
}
```
```json
{
  "data": {
    "node": {
      "items": {
        "pageInfo": {
          "hasNextPage": false
        },
        "edges": [
          {
            "node": {
              "content": {
                "title": "Investigate github project cli",
                "id": "DI_...",
                "assignees": {
                  "pageInfo": {
                    "hasNextPage": false
                  },
                  "edges": [
                    {
                      "node": {
                        "login": "USERNAME"
                      }
                    }
                  ]
                }
              },
              "id": "PVTI_...",
              "type": "DRAFT_ISSUE"
            }
          },

          ...
```
