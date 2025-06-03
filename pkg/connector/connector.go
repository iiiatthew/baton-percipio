package connector

import (
	"context"
	"io"
	"time"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type Connector struct {
	client       *client.Client
	limitCourses mapset.Set[string]
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (d *Connector) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	logger := ctxzap.Extract(ctx)
	startTime := time.Now()

	logger.Info("SYNC STARTED",
		zap.Time("timestamp", startTime),
		zap.String("connector", "baton-percipio"),
	)

	defer func() {
		logger.Info("SYNC COMPLETED",
			zap.Time("timestamp", time.Now()),
			zap.Duration("duration", time.Since(startTime)),
		)
	}()

	// Log resource syncer configuration
	limitCoursesCount := 0
	if d.limitCourses != nil {
		limitCoursesCount = d.limitCourses.Cardinality()
	}

	logger.Info("Sync configuration",
		zap.Int("limitCoursesCount", limitCoursesCount),
		zap.Bool("courseLimitingEnabled", d.limitCourses != nil),
	)

	return []connectorbuilder.ResourceSyncer{
		newUserBuilder(d.client),
		newCourseBuilder(d.client, d.limitCourses),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (d *Connector) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (d *Connector) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Percipio Connector",
		Description: "Connector syncing users from Percipio",
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (d *Connector) Validate(ctx context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New returns a new instance of the connector.
func New(
	ctx context.Context,
	organizationID string,
	token string,
	limitCourses []string,
) (*Connector, error) {
	percipioClient, err := client.New(
		ctx,
		client.BaseApiUrl,
		organizationID,
		token,
	)
	if err != nil {
		return nil, err
	}

	connector := &Connector{
		client: percipioClient,
	}

	if len(limitCourses) > 0 {
		connector.limitCourses = mapset.NewSet(limitCourses...)
	}

	return connector, nil
}
