package health

import (
	"context"
	"net/http"
	"time"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	pinger func(ctx context.Context) error
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	var pinger func(ctx context.Context) error
	if db != nil {
		pinger = func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		}
	}
	return &HealthHandler{pinger: pinger}
}

func NewHealthHandlerWithPinger(pinger func(ctx context.Context) error) *HealthHandler {
	return &HealthHandler{pinger: pinger}
}

// Check returns server health status and database connectivity.
//
//	@Summary		Health check
//	@Description	Check server health status and system database connectivity.
//	@Tags			Health
//	@Produce		json
//	@Success		200	{object}	response.BaseResponse[HealthResponse]	"Service is healthy"
//	@Failure		503	{object}	response.BaseResponse[HealthResponse]	"Service is degraded"
//	@Router			/health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	services := make(map[string]string)
	status := "ok"
	statusCode := http.StatusOK

	if h.pinger != nil {
		if err := h.pinger(c.Request.Context()); err != nil {
			services["database"] = "down"
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		} else {
			services["database"] = "up"
		}
	}

	resp := HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  services,
	}

	if statusCode == http.StatusOK {
		c.JSON(statusCode, response.Success("service is healthy", resp))
		return
	}

	c.JSON(statusCode, response.BaseResponse[HealthResponse]{
		Success: false,
		Message: "service is degraded",
		Data:    resp,
	})
}
