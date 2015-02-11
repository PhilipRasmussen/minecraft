package profile

import (
	"errors"
	"fmt"
)

/******************
* EXPORTED ERRORS *
******************/

// An ErrNoSuchUser error signals that an operation failed because no profile exists with the denoted username.
type ErrNoSuchUser struct {

	// The username for which no profile could be found
	Name string
}

func (e ErrNoSuchUser) Error() string {

	return fmt.Sprintf("user %s: no such profile", e.Name)
}

// /////////////

// An ErrNoSuchUUID error signals that an operation failed because no profile exists with the denoted UUID.
type ErrNoSuchUUID struct {

	// The UUID for which no profile could be found
	UUID string
}

func (e ErrNoSuchUUID) Error() string {

	return fmt.Sprintf("UUID %s: no such profile", e.UUID)
}

// /////////////

// An ErrTooManyRequests error occurs when the client has exceeded its server communication rate limit.
// At the time of writing, the load operations have a shared rate limit of 600 requests per 10 minutes.
type ErrTooManyRequests string

func (e ErrTooManyRequests) Error() string {

	return string(e)
}

var errTooManyRequests ErrTooManyRequests = "request rate limit exceeded"

// /////////////

// An ErrMaxSizeExceeded error occurs when LoadMany is requested to load more than LoadManyMaxSize profiles at once.
type ErrMaxSizeExceeded struct {

	// The number of profiles which were requested
	Size int
}

func (e ErrMaxSizeExceeded) Error() string {

	return fmt.Sprintf("aggregate request size of %d exceeded maximum of %d", e.Size, LoadManyMaxSize)
}

// /////////////

// Used by LoadMany to call buildProfile to exclude demo profiles from its results
var errDemo = errors.New("demo profile detected")

/************
* INTERNALS *
************/

// Extracts any Mojang error from a piece of JSON decoded using the encoding/json package
// Mojang errors are JSON objects with "error" and "errorMessage" fields
func getJsonError(json interface{}) error {

	if m, isMap := json.(map[string]interface{}); isMap {

		if e, failed := m["error"]; failed {

			error := e.(string)
			switch error {

			case "TooManyRequestsException":
				return errTooManyRequests

			default:
				const errMsg = "Mojang API error: %s; message: %s"
				return errors.New(fmt.Sprintf(errMsg, error, m["errorMessage"].(string)))
			}
		}
	}

	return nil
}
