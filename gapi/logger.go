package gapi

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


func GrpcLogger(
	ctx context.Context, 
	req any, 
	info *grpc.UnaryServerInfo, 
	handler grpc.UnaryHandler,
	) (resp any, err error) {
		startTime := time.Now()
		result, err := handler(ctx, req)
		duration := time.Since(startTime)

		statusCode := codes.Unknown
		if st, ok := status.FromError(err); ok {
			statusCode = st.Code()
		}

		logger := log.Info()
		if err != nil {
			logger = log.Error().Err(err)
		}

		logger.Str("protocol", "grpc").
			Str("method", info.FullMethod).
			Int("status_code", int(statusCode)).
			Str("status_text", statusCode.String()).
			Dur("duration", duration).
			Msg("received a gRPC request")

		return result, err
}	

// ResponseRecorder wraps an http.ResponseWriter to record the HTTP status code.
type ResponseRecorder struct {
    http.ResponseWriter
    StatusCode int // Captures the status code written via WriteHeader
	Body []byte
}

// WriteHeader records the status code and delegates to the underlying ResponseWriter.
func (rec *ResponseRecorder) WriteHeader(statusCode int) {
    rec.StatusCode = statusCode
    rec.ResponseWriter.WriteHeader(statusCode)
}

func (rec *ResponseRecorder) Write(body []byte) (int, error) {
	rec.Body = body
	return rec.ResponseWriter.Write(body)
}
// HttpLogger wraps an http.Handler and logs request details with duration and status.
func HttpLogger(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        startTime := time.Now()

        // Wrap the ResponseWriter to capture status code
        rec := &ResponseRecorder{
            ResponseWriter: res,
            StatusCode:     http.StatusOK, // default
        }

        // Serve the request
        handler.ServeHTTP(rec, req)

        // Calculate duration
        duration := time.Since(startTime)

		logger := log.Info()
		if rec.StatusCode != http.StatusOK {
			logger = log.Error().Bytes("body", rec.Body)
		}

        // Log request info
        logger.Str("protocol", "http").
            Str("method", req.Method).
            Str("path", req.RequestURI).
            Int("status_code", rec.StatusCode).
            Str("status_text", http.StatusText(rec.StatusCode)). // ✅ Fixed: use http.StatusText()
            Dur("duration", duration)

        logger.Msg("received HTTP request")
    })
}