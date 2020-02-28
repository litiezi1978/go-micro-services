package tracing

import (
	"fmt"
	"io"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

func Init(serviceName, host string) (opentracing.Tracer, io.Closer, error) {
	cfg := config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "probabilistic",
			Param: 0.3,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  host,
		},
	}

	tracer, closer, err := cfg.New(serviceName)
	if err != nil {
		return nil, nil, fmt.Errorf("Creating tracer error: %v", err)
	}
	return tracer, closer, nil
}
