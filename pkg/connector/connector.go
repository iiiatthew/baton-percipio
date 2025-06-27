package connector

import (
	"context"
	"io"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	mapset "github.com/deckarep/golang-set/v2"
)

// Connector struct is the main entry point for the Percipio connector.
// It is defined by the baton-sdk and is responsible for managing the connector's state.
// It holds the API client and a set of course IDs to limit the sync scope.
// This structure organizes the connector's dependencies and configuration.
// Instances are created by the New function with configuration provided at startup.
type Connector struct {
	client       *client.Client
	limitCourses mapset.Set[string]
}

// ResourceSyncers method returns a list of resource syncers for the connector.
// It implements the `ResourceSyncers` method required by the `connectorbuilder.Connector` interface.
// The method initializes and returns a `ResourceSyncer` for each resource type (users and courses) that the connector should sync.
// Which provides the baton-sdk with the necessary builders to handle the synchronization of each resource type.
// This implementation returns a fixed list containing a user builder and a course builder.
func (d *Connector) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		newUserBuilder(d.client),
		newCourseBuilder(d.client, d.limitCourses),
	}
}

// Asset method is a placeholder for asset fetching functionality.
// It implements the `Asset` method required by the `connectorbuilder.Connector` interface.
// The method is not implemented in this connector.
// Which is a required part of the connector interface but is not needed for this connector's operation.
// This implementation currently returns nil and is not used.
func (d *Connector) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata method returns descriptive metadata about the connector.
// It implements the `Metadata` method required by the `connectorbuilder.Connector` interface.
// The method returns a `ConnectorMetadata` object containing the display name and description.
// Which provides the Baton application with information about the connector's purpose.
// This implementation returns a static object with pre-defined text.
func (d *Connector) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Percipio Connector",
		Description: "Connector syncing users from Percipio",
	}, nil
}

// Validate method is a placeholder for configuration validation.
// It implements the `Validate` method required by the `connectorbuilder.Connector` interface.
// The method is intended to ensure that the connector is properly configured by exercising API credentials.
// Which allows the Baton application to verify that the provided configuration is valid before starting a sync.
// This implementation currently returns nil and does not perform any validation.
func (d *Connector) Validate(ctx context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New function creates and initializes a new Percipio Connector.
// It implements the constructor required by the main application to start the connector.
// The function initializes a new Percipio API client and constructs the `Connector` struct with the client and any course limitations.
// Which provides a fully configured instance of the connector, ready to be used by the baton-sdk.
// This implementation uses `mapset` to efficiently store and check for limited courses if they are provided.
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
