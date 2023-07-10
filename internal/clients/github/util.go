package github

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

/*
GqlErrorStringOr tries to convert an error that came from a GraphQL query into a user-understandable string. fmtStr is
the first parameter to fmt.Sprintf and the error `string` is the only other parameter.

If the error cannot be classified (and thus prettified) returns `ifNotGqlError`.

The function should be called only `if err != nil`. If the error is nil the function panics (indicating that you should
find where you called it without checking and fix that).
*/
func GqlErrorStringOr(fmtStr string, err error, ifNotGqlError string) string {
	if err == nil {
		panic("github.GqlErrorStringOr() expects an `error != nil`")
	}

	var gqlerr *gqlerror.Error
	if errors.As(err, &gqlerr) {
		return fmt.Sprint(fmtStr, gqlerr.Error())
	}

	var gqllist *gqlerror.List
	if errors.As(err, &gqllist) {
		return fmt.Sprint(fmtStr, gqllist.Error())
	}

	return ifNotGqlError
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
