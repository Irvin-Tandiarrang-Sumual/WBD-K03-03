package handlers

import (
	"context"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct {
	handler http.Handler
	refresh []func(context.Context) error
}

func NewMetricsHandler(refresh ...func(context.Context) error) *MetricsHandler {
	return &MetricsHandler{
		refresh: refresh,
		handler: promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}),
	}
}

func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	for _, refresh := range h.refresh {
		if err := refresh(c.Request.Context()); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
	}
	h.handler.ServeHTTP(c.Writer, c.Request)
}
