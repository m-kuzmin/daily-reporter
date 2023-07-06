package github

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

/*
GqlErrorString returns `gqlerror.Error.Message` if the error is of that type. If the error is nil or not from gql then
`"", false`.
*/
func GqlErrorString(err error) (string, bool) {
	if err != nil {
		var gqlerr *gqlerror.Error
		if errors.As(err, &gqlerr) {
			return gqlerr.Message, true
		}
	}

	return "", false
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

type EmptyResponseError struct {
	Message string
}

func (e EmptyResponseError) Error() string {
	return fmt.Sprintf("we expected something from GitHub, but it gave us nothing. details: %s", e.Message)
}
