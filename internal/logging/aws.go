package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var requiredCloudwatchEnvs = []string{
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_REGION",
}

type CloudwatchHandler struct {
	client        *cloudwatchlogs.Client
	logGroupName  string
	logStreamName string
	sequenceToken *string
	mu            sync.Mutex
	attrs         []slog.Attr
	groups        []string
	logLevel      slog.Level
}

type CloudwatchHandlerParams struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	GroupName       string
	StreamName      string
	LogLevel        slog.Level
	RetentionDays   int32 // Number of days to retain logs (0 = never expire)
}

func NewCloudwatchLogger(ctx context.Context, params CloudwatchHandlerParams) (*CloudwatchHandler, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(params.Region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				params.AccessKeyID,
				params.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(cfg)

	// create log group if it doesn't exist
	_, err = client.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(params.GroupName),
	})
	if err != nil {
		slog.Info("cloudwatch log group already exists", "group_name", params.GroupName)
	}

	// set retention policy if specified
	if params.RetentionDays > 0 {
		_, err = client.PutRetentionPolicy(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
			LogGroupName:    aws.String(params.GroupName),
			RetentionInDays: aws.Int32(params.RetentionDays),
		})
		if err != nil {
			slog.Warn("failed to set retention policy", "error", err.Error())
		} else {
			slog.Info("cloudwatch retention policy set", "days", params.RetentionDays)
		}
	}

	// create log stream with the current timestamp
	streamTime := fmt.Sprintf("%s-%s", params.StreamName, time.Now().Format("2006-01-02-15-04-05"))
	_, err = client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(params.GroupName),
		LogStreamName: aws.String(streamTime),
	})
	if err != nil {
		slog.Error("creating log stream", "group_name", params.GroupName, "stream_name", streamTime)
	} else {
		slog.Info("created log stream", "group_name", params.GroupName, "stream_name", streamTime)
	}

	return &CloudwatchHandler{
		client:        client,
		logGroupName:  params.GroupName,
		logStreamName: streamTime,
		logLevel:      params.LogLevel,
	}, nil
}

func (h *CloudwatchHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.logLevel
}

func (h *CloudwatchHandler) Handle(ctx context.Context, record slog.Record) error {
	logEntry := map[string]any{
		"time":    record.Time.Format(time.RFC3339),
		"level":   record.Level.String(),
		"message": record.Message,
	}

	for _, attr := range h.attrs {
		addAttrToMap(logEntry, attr)
	}

	record.Attrs(func(attr slog.Attr) bool {
		addAttrToMap(logEntry, attr)
		return true
	})

	if len(h.groups) > 0 {
		logEntry["groups"] = h.groups
	}

	message, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	input := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(h.logGroupName),
		LogStreamName: aws.String(h.logStreamName),
		LogEvents: []types.InputLogEvent{
			{
				Message:   aws.String(string(message)),
				Timestamp: aws.Int64(record.Time.UnixMilli()),
			},
		},
	}

	if h.sequenceToken != nil {
		input.SequenceToken = h.sequenceToken
	}

	result, err := h.client.PutLogEvents(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put log events: %w", err)
	}

	h.sequenceToken = result.NextSequenceToken
	return nil
}

func (h *CloudwatchHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &CloudwatchHandler{
		client:        h.client,
		logGroupName:  h.logGroupName,
		logStreamName: h.logStreamName,
		sequenceToken: h.sequenceToken,
		attrs:         newAttrs,
		groups:        h.groups,
	}
}

func (h *CloudwatchHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &CloudwatchHandler{
		client:        h.client,
		logGroupName:  h.logGroupName,
		logStreamName: h.logStreamName,
		sequenceToken: h.sequenceToken,
		attrs:         h.attrs,
		groups:        newGroups,
	}
}

func GetCloudwatchParamsFromEnv() CloudwatchHandlerParams {
	p := &CloudwatchHandlerParams{
		Region:          os.Getenv("AWS_REGION"),
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		GroupName:       "lightsail/ticketbot",
		StreamName:      "container-logs",
		LogLevel:        slog.LevelInfo,
		RetentionDays:   7,
	}

	if groupEnv := os.Getenv("CLOUDWATCH_GROUP_NAME"); groupEnv != "" {
		p.GroupName = groupEnv
	}

	if streamEnv := os.Getenv("CLOUDWATCH_STREAM_NAME"); streamEnv != "" {
		p.StreamName = streamEnv
	}

	if retentionEnv := os.Getenv("CLOUDWATCH_RETENTION_DAYS"); retentionEnv != "" {
		if days, err := strconv.Atoi(retentionEnv); err == nil {
			p.RetentionDays = int32(days)
		}
	}

	if os.Getenv("DEBUG") == "true" {
		p.LogLevel = slog.LevelDebug
	}

	return *p
}

func CloudwatchVarsSet() bool {
	for _, v := range requiredCloudwatchEnvs {
		if os.Getenv(v) == "" {
			return false
		}
	}
	return true
}

func addAttrToMap(m map[string]any, attr slog.Attr) {
	if attr.Value.Kind() == slog.KindGroup {
		// Handle group attributes by creating a nested map
		groupMap := make(map[string]any)
		for _, groupAttr := range attr.Value.Group() {
			addAttrToMap(groupMap, groupAttr)
		}
		m[attr.Key] = groupMap
	} else {
		val := attr.Value.Any()
		if err, ok := val.(error); ok {
			m[attr.Key] = err.Error()
		} else {
			m[attr.Key] = val
		}
		m[attr.Key] = attr.Value.Any()
	}
}
