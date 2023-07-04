# Obtaining a list of project items

The returned list contains
- Title of the item
- Status or `null`
- If is assigned to viewer (owner of API token)

```graphql
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
```

```json
{
  "data": {
    "node": {
      "items": {
        "nodes": [
          {
            "status": {
              "name": "In Progress"
            },
            "assignedTo": {
              "users": {
                "nodes": [
                  {
                    "isViewer": true
                  }
                ]
              }
            },
            "content": {
              "title": "/dailyStatus"
            }
          }
        ],
        "pageInfo": {
          "endCursor": "MQ",
          "startCursor": "MQ",
          "hasNextPage": true
        }
      }
    }
  }
}
```
