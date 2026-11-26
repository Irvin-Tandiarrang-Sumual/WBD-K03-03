package handlers

import (
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsHandler struct {
	handler http.Handler
}

func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		handler: promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}),
	}
}

func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	h.handler.ServeHTTP(c.Writer, c.Request)
}
