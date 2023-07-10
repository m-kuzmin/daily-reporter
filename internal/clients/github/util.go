package github

import (
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
