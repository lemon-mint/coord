package anthropic

import "github.com/lemon-mint/coord/llm"

func getErrorByType(err_t string) error {
	switch err_t {
	case "invalid_request_error":
		return llm.ErrInvalidRequest
	case "authentication_error":
		return llm.ErrAuthentication
	case "permission_error":
		return llm.ErrPermission
	case "not_found_error":
		return llm.ErrNotFound
	case "rate_limit_error":
		return llm.ErrRateLimit
	case "api_error":
		return llm.ErrInternalServer
	case "overloaded_error":
		return llm.ErrOverloaded
	}

	return llm.ErrUnknown
}

func getErrorByStatus(err_c int) error {
	switch err_c {
	case 400:
		// invalid_request_error
		return llm.ErrInvalidRequest
	case 401:
		// authentication_error
		return llm.ErrAuthentication
	case 403:
		// permission_error
		return llm.ErrPermission
	case 404:
		// not_found_error
		return llm.ErrNotFound
	case 429:
		// rate_limit_error
		return llm.ErrRateLimit
	case 500:
		// api_error
		return llm.ErrInternalServer
	case 529:
		// overloaded_error
		return llm.ErrOverloaded
	}
	return llm.ErrUnknown
}
