package krakend

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	metrics "github.com/krakendio/krakend-metrics/v2/gin"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
)

// SSEConfig holds the configuration for SSE endpoints
type SSEConfig struct {
	KeepAliveInterval time.Duration `json:"keep_alive_interval"`
	RetryInterval     int           `json:"retry_interval"`
}

// SSEHandlerFactory creates handlers for SSE endpoints
type SSEHandlerFactory struct {
	logger  logging.Logger
	metrics *metrics.Metrics
}

// NewSSEHandlerFactory returns a new SSEHandlerFactory
func NewSSEHandlerFactory(logger logging.Logger, metrics *metrics.Metrics) *SSEHandlerFactory {
	return &SSEHandlerFactory{
		logger:  logger,
		metrics: metrics,
	}
}

// NewHandler creates a new SSE handler
func (s *SSEHandlerFactory) NewHandler(cfg *config.EndpointConfig, prxy proxy.Proxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set SSE headers
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// Get SSE config
		var sseCfg SSEConfig
		if v, ok := cfg.ExtraConfig["sse"]; ok && v != nil {
			if b, err := json.Marshal(v); err == nil {
				json.Unmarshal(b, &sseCfg)
			}
		}

		// Set default values
		if sseCfg.KeepAliveInterval == 0 {
			sseCfg.KeepAliveInterval = 30 * time.Second
		}
		if sseCfg.RetryInterval == 0 {
			sseCfg.RetryInterval = 1000
		}

		// Send retry interval
		c.Writer.Write([]byte("retry: " + strconv.Itoa(sseCfg.RetryInterval) + "\n\n"))
		c.Writer.Flush()

		// Create context with cancel
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		// Start keep-alive goroutine
		go func() {
			ticker := time.NewTicker(sseCfg.KeepAliveInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					c.Writer.Write([]byte(": keepalive\n\n"))
					c.Writer.Flush()
				case <-ctx.Done():
					return
				}
			}
		}()

		// Create a channel to receive proxy responses
		responseChan := make(chan *proxy.Response, 1)
		errorChan := make(chan error, 1)

		// Execute proxy in a goroutine
		go func() {
			response, err := prxy(ctx, &proxy.Request{
				Method:  c.Request.Method,
				URL:     c.Request.URL,
				Headers: c.Request.Header,
				Body:    c.Request.Body,
			})
			if err != nil {
				errorChan <- err
				return
			}
			responseChan <- response
		}()

		// Handle responses
		for {
			select {
			case err := <-errorChan:
				s.logger.Error("SSE proxy error:", err)
				return
			case response := <-responseChan:
				if response == nil || response.Data == nil {
					continue
				}
				// Stream the response data
				if data, err := json.Marshal(response.Data); err == nil {
					c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
					c.Writer.Flush()
				}
			case <-ctx.Done():
				return
			}
		}
	}
}
