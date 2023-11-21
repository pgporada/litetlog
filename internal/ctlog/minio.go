package ctlog

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MinioBackend struct {
	getClient *minio.Client
	putClient *minio.Client
	bucket    string
	metrics   []prometheus.Collector
}

func NewMinioBackend(ctx context.Context, region, bucket string) (*MinioBackend, error) {
	// Phil
	endpoint := "127.0.0.1:9000"
	accessKeyID := "TAXO0RQKdakH6Z38GO7d"
	secretAccessKey := "mM1kCKqRZ5v3Xmzf4oJjC6Da7B35gv9RXfU8gZ8b"
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	// Phil

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "minio_requests_total",
		},
		[]string{"action", "code"},
	)
	duration := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "minio_request_duration_seconds",
			Objectives: map[float64]float64{0.5: 0.05, 0.75: 0.025, 0.9: 0.01, 0.99: 0.001},
			MaxAge:     1 * time.Minute,
			AgeBuckets: 6,
		},
		[]string{"action", "code"},
	)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	getLabels := prometheus.Labels{"action": "get"}
	getTransport := http.RoundTripper(http.DefaultTransport.(*http.Transport).Clone())
	getTransport = promhttp.InstrumentRoundTripperCounter(counter.MustCurryWith(getLabels), getTransport)
	getTransport = promhttp.InstrumentRoundTripperDuration(duration.MustCurryWith(getLabels), getTransport)
	getCfg := cfg.Copy()
	getCfg.HTTPClient = &http.Client{Transport: getTransport}

	putLabels := prometheus.Labels{"action": "put"}
	putTransport := http.RoundTripper(http.DefaultTransport.(*http.Transport).Clone())
	putTransport = promhttp.InstrumentRoundTripperCounter(counter.MustCurryWith(putLabels), putTransport)
	putTransport = promhttp.InstrumentRoundTripperDuration(duration.MustCurryWith(putLabels), putTransport)
	putCfg := cfg.Copy()
	putCfg.HTTPClient = &http.Client{Transport: putTransport}

	return &MinioBackend{
		getClient: minioClient,
		putClient: minioClient,
		bucket:    bucket,
		metrics:   []prometheus.Collector{counter, duration},
	}, nil
}

var _ Backend = &MinioBackend{}

func (m *MinioBackend) Upload(ctx context.Context, key string, data []byte) error {
	// TODO: give up on slow requests and retry.
	_, err := m.putClient.PutObject(ctx,
		m.bucket,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{},
	)
	return err
}

func (m *MinioBackend) Fetch(ctx context.Context, key string) ([]byte, error) {
	out, err := m.getClient.GetObject(ctx,
		m.bucket,
		key,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	return io.ReadAll(out)
}

func (m *MinioBackend) Metrics() []prometheus.Collector {
	return m.metrics
}
