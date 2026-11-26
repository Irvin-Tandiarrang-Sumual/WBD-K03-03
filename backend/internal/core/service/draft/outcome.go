package draft

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

func outcomeUnknown(err error) bool {
	if errors.Is(err, domain.ErrMigrationOutcomeUnknown) || errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, driver.ErrBadConn) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var state interface{ SQLState() string }
	if errors.As(err, &state) && strings.HasPrefix(state.SQLState(), "08") {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "connection reset") || strings.Contains(message, "broken pipe")
}
