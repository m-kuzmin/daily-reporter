# Get user's v2 projects

```graphql
{
  viewer {
    projectsV2(first: 10) {
      edges {
        node {
          id
          title
        }
      }
    }
  }
}
```
```json
{
  "data": {
    "viewer": {
      "projectsV2": {
        "edges": [
          {
            "node": {
              "id": "PVT_...",
              "title": "Token test project"
            }
          },
...
```
