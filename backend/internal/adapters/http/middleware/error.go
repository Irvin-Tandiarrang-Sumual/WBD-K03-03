package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		rawErr := c.Errors.Last().Err
		statusCode, clientBody := MapError(rawErr)
		c.JSON(statusCode, clientBody)
	}
}

func MapError(err error) (int, response.BaseResponse[any]) {
	if err == nil {
		return http.StatusOK, response.Success[any]("", nil)
	}

	switch {
	case isBindingOrJSONError(err):
		return http.StatusBadRequest, response.Error(err.Error())

	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, response.Error("invalid username or password")
	case errors.Is(err, domain.ErrInvalidSession):
		return http.StatusUnauthorized, response.Error("invalid or expired session")
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		return http.StatusUnauthorized, response.Error("authentication required")

	case errors.Is(err, domain.ErrAccessDenied),
		errors.Is(err, domain.ErrDatabaseAccessDenied),
		errors.Is(err, domain.ErrInvalidCSRFToken):
		return http.StatusForbidden, response.Error(err.Error())

	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, response.Error(err.Error())
	case errors.Is(err, domain.ErrGroupNotFound):
		return http.StatusNotFound, response.Error(err.Error())
	case errors.Is(err, domain.ErrDatabaseNotFound):
		return http.StatusNotFound, response.Error(err.Error())
	case errors.Is(err, domain.ErrMigrationNotFound):
		return http.StatusNotFound, response.Error(err.Error())

	case errors.Is(err, domain.ErrUsernameTaken),
		errors.Is(err, domain.ErrGroupNameTaken),
		errors.Is(err, domain.ErrDatabaseAlreadyExists),
		errors.Is(err, domain.ErrDatabaseInUse),
		errors.Is(err, domain.ErrMembershipExists),
		errors.Is(err, domain.ErrMembershipNotFound),
		errors.Is(err, domain.ErrGroupMustHaveDatabase),
		errors.Is(err, domain.ErrUserMustHaveGroup),
		errors.Is(err, domain.ErrLastAdmin),
		errors.Is(err, domain.ErrLeaderRequiresReplacement),
		errors.Is(err, domain.ErrLeaderMustBeGroupMember),
		errors.Is(err, domain.ErrUserIsGroupLeader),
		errors.Is(err, domain.ErrMigrationRollbackUnavailable),
		errors.Is(err, domain.ErrMigrationVersionUnavailable),
		errors.Is(err, domain.ErrMigrationInProgress):
		return http.StatusConflict, response.Error(err.Error())

	case errors.Is(err, domain.ErrPasswordTooLong):
		return http.StatusBadRequest, response.Error("password must not exceed 72 bytes")
	case errors.Is(err, domain.ErrPasswordEmpty):
		return http.StatusBadRequest, response.Error("password cannot be empty")
	case errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrInvalidPagination),
		errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidGroupsIDs),
		errors.Is(err, domain.ErrInvalidDatabaseAccess),
		errors.Is(err, domain.ErrInvalidDatabaseInput),
		errors.Is(err, domain.ErrInvalidDatabaseQuery),
		errors.Is(err, domain.ErrInvalidMigrationInput),
		errors.Is(err, domain.ErrInvalidMigrationComparison),
		errors.Is(err, domain.ErrSQLDownGenerationUnsupported),
		errors.Is(err, domain.ErrDatabaseConnectionFailed):
		return http.StatusBadRequest, response.Error(err.Error())

	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, response.Error("invalid input")

	case errors.Is(err, domain.ErrQueryTimeout),
		errors.Is(err, context.DeadlineExceeded):
		return http.StatusRequestTimeout, response.Error("query execution timed out")

	default:
		errLower := strings.ToLower(err.Error())
		if isDatabaseConnectionError(errLower) {
			return http.StatusBadRequest, response.Error("database connection failed")
		}
		return http.StatusInternalServerError, response.Error("internal server error")
	}
}

func isBindingOrJSONError(err error) bool {
	var syntaxErr *json.SyntaxError
	var unmarshalTypeErr *json.UnmarshalTypeError
	var validationErrs validator.ValidationErrors

	if errors.As(err, &syntaxErr) ||
		errors.As(err, &unmarshalTypeErr) ||
		errors.As(err, &validationErrs) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "json:") ||
		strings.Contains(errStr, "unmarshal") ||
		strings.Contains(errStr, "cannot unmarshal") ||
		strings.Contains(errStr, "unexpected eof")
}

func isDatabaseConnectionError(errLower string) bool {
	return strings.Contains(errLower, "connection refused") ||
		strings.Contains(errLower, "no such host") ||
		strings.Contains(errLower, "dial tcp") ||
		strings.Contains(errLower, "failed to connect") ||
		strings.Contains(errLower, "authentication failed") ||
		strings.Contains(errLower, "access denied for user")
}
