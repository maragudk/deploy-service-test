package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/logging"
	"github.com/honeybadger-io/honeybadger-go"
	"github.com/maragudk/aws/s3"
	"github.com/maragudk/env"
	"github.com/maragudk/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"golang.org/x/sync/errgroup"

	"github.com/maragudk/service/email"
	"github.com/maragudk/service/http"
	"github.com/maragudk/service/jobs"
	"github.com/maragudk/service/sql"
)

func main() {
	os.Exit(start())
}

func start() int {
	log := log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile|log.LUTC)
	log.Println("Starting")

	_ = env.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	honeybadgerAPIKey := env.GetStringOrDefault("HONEYBADGER_API_KEY", "")
	if honeybadgerAPIKey != "" {
		honeybadger.Configure(honeybadger.Configuration{
			APIKey: honeybadgerAPIKey,
			Env:    "production",
			Logger: log,
		})

		defer honeybadger.Flush()
		defer honeybadger.Monitor()
	} else {
		honeybadger.Configure(honeybadger.Configuration{
			Backend: honeybadger.NewNullBackend(),
			Env:     "development",
			Logger:  log,
		})
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(collectors.NewGoCollector())

	db := sql.NewDatabase(sql.NewDatabaseOptions{
		Log:                   log,
		Metrics:               registry,
		URL:                   env.GetStringOrDefault("DATABASE_URL", "app.db"),
		MaxOpenConnections:    env.GetIntOrDefault("DATABASE_MAX_OPEN_CONNS", 5),
		MaxIdleConnections:    env.GetIntOrDefault("DATABASE_MAX_IDLE_CONNS", 5),
		ConnectionMaxLifetime: env.GetDurationOrDefault("DATABASE_CONN_MAX_LIFETIME", time.Hour),
		ConnectionMaxIdleTime: env.GetDurationOrDefault("DATABASE_CONN_MAX_IDLE_TIME", time.Hour),
	})

	if err := db.Connect(); err != nil {
		log.Println("Error connecting to database:", err)
		return 1
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithLogger(createAWSLogAdapter(log)),
		config.WithEndpointResolverWithOptions(createAWSEndpointResolver()),
	)
	if err != nil {
		log.Println("Error creating AWS config:", err)
		return 1
	}

	var bucket *s3.Bucket
	bucketName := env.GetStringOrDefault("S3_BUCKET_NAME", "")
	if bucketName != "" {
		bucket = s3.NewBucket(s3.NewBucketOptions{
			Config:    awsConfig,
			Name:      bucketName,
			PathStyle: env.GetBoolOrDefault("S3_PATH_STYLE", false),
		})
	}

	emailSender := email.NewSender(email.NewSenderOptions{
		BaseURL:                   env.GetStringOrDefault("BASE_URL", "http://localhost:8080"),
		Log:                       log,
		MarketingEmailAddress:     env.GetStringOrDefault("MARKETING_EMAIL_ADDRESS", "marketing@example.com"),
		MarketingEmailName:        env.GetStringOrDefault("MARKETING_EMAIL_NAME", "Marketing"),
		Metrics:                   registry,
		ReplyToEmailAddress:       env.GetStringOrDefault("REPLY_TO_EMAIL_ADDRESS", "support@example.com"),
		ReplyToEmailName:          env.GetStringOrDefault("REPLY_TO_EMAIL_NAME", "Support"),
		Token:                     env.GetStringOrDefault("POSTMARK_TOKEN", ""),
		TransactionalEmailAddress: env.GetStringOrDefault("TRANSACTIONAL_EMAIL_ADDRESS", "transactional@example.com"),
		TransactionalEmailName:    env.GetStringOrDefault("TRANSACTIONAL_EMAIL_NAME", "Transactional"),
	})

	s := http.NewServer(http.NewServerOptions{
		AdminPassword: env.GetStringOrDefault("ADMIN_PASSWORD", "08f439unf398nf92oiwfoif3oiewifmowe"),
		Database:      db,
		Host:          env.GetStringOrDefault("HOST", ""),
		Log:           log,
		Metrics:       registry,
		Bucket:        bucket,
		Port:          env.GetIntOrDefault("PORT", 8080),
		SecureCookie:  env.GetBoolOrDefault("SECURE_COOKIE", true),
	})

	runner := jobs.NewRunner(jobs.NewRunnerOptions{
		Database:     db,
		EmailSender:  emailSender,
		JobLimit:     5,
		Log:          log,
		Metrics:      registry,
		PollInterval: time.Second,
		Queue:        db,
	})

	eg, ctx := errgroup.WithContext(ctx)

	if env.GetBoolOrDefault("SERVER_ENABLED", true) {
		eg.Go(func() error {
			if err := s.Start(); err != nil {
				return errors.Wrap(err, "error starting server")
			}
			return nil
		})
	}

	if env.GetBoolOrDefault("JOBS_ENABLED", true) {
		eg.Go(func() error {
			runner.Start(ctx)
			return nil
		})
	}

	<-ctx.Done()
	log.Println("Stopping")

	if env.GetBoolOrDefault("SERVER_ENABLED", true) {
		eg.Go(func() error {
			if err := s.Stop(); err != nil {
				return errors.Wrap(err, "error stopping server")
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Println("Error:", err)
		return 1
	}

	log.Println("Stopped")

	return 0
}

func createAWSLogAdapter(log *log.Logger) logging.LoggerFunc {
	return func(classification logging.Classification, format string, v ...any) {
		log.Printf(string(classification)+" "+format, v...)
	}
}

// createAWSEndpointResolver used for local development endpoints.
// See https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/endpoints/
func createAWSEndpointResolver() aws.EndpointResolverWithOptionsFunc {
	s3EndpointURL := env.GetStringOrDefault("S3_ENDPOINT_URL", "")

	return func(service, region string, options ...any) (aws.Endpoint, error) {
		switch service {
		case awss3.ServiceID:
			if s3EndpointURL != "" {
				return aws.Endpoint{
					URL: s3EndpointURL,
				}, nil
			}
		}
		// Fallback to default endpoint
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	}
}
