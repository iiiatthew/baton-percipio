package connector

import (
	"context"
	"strings"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	userFullNameDefault = "<no name>"
)

// userBuilder struct is responsible for syncing user resources.
// It is used by the connector to fetch and process user data from the Percipio API.
// It holds a reference to the API client and the user resource type descriptor.
// This structure organizes the context needed for user synchronization operations.
// Instances are created by the `newUserBuilder` function.
type userBuilder struct {
	client       *client.Client
	resourceType *v2.ResourceType
}

// ResourceType method returns the resource type descriptor for users.
// It implements the `ResourceType` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method returns the static `userResourceType` object defined for this connector.
// Which informs the baton-sdk about the type of resource this syncer is responsible for.
// This implementation returns a pre-defined object.
func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return o.resourceType
}

// getDisplayName function constructs a display name for a user.
// It implements the name generation logic for creating user resources.
// The function concatenates the user's first and last name, falling back to email or ID if names are not available.
// Which ensures that every user resource has a human-readable display name.
// This implementation provides a consistent naming convention with sensible fallbacks.
func getDisplayName(user client.User) string {
	var parts []string
	if user.FirstName != "" {
		parts = append(parts, user.FirstName)
	}
	if user.LastName != "" {
		parts = append(parts, user.LastName)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if user.Email != "" {
		return user.Email
	}
	if user.Id != "" {
		return user.Id
	}
	return userFullNameDefault
}

// userResource function creates a new `v2.Resource` from a Percipio user object.
// It implements the mapping from the provider's user data model to the baton-sdk's resource model.
// The function populates the resource's profile with user attributes and sets the user trait with email, status, and login information.
// Which is the core transformation for converting raw user data into a standardized resource object that Baton can process.
// This implementation uses the `resourceSdk.NewUserResource` helper to construct a resource with the `UserTrait`.
func userResource(user client.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"id":                                user.Id,
		"external_id":                       user.ExternalUserId,
		"approval_manager":                  user.ApprovalManager.Email,
		"email":                             user.Email,
		"first_name":                        user.FirstName,
		"has_coaching":                      user.HasCoaching,
		"has_enterprise_coaching":           user.HasEnterpriseCoaching,
		"has_enterprise_coaching_dashboard": user.HasEnterpriseCoaching,
		"is_active":                         user.IsActive,
		"is_instructor":                     user.IsInstructor,
		"job_title":                         user.JobTitle,
		"last_name":                         user.LastName,
		"login_name":                        user.LoginName,
		"role":                              user.Role,
	}

	for _, attribute := range user.CustomAttributes {
		profile[attribute.Name] = attribute.Value
	}

	status := v2.UserTrait_Status_STATUS_DISABLED
	if user.IsActive {
		status = v2.UserTrait_Status_STATUS_ENABLED
	}

	userTraitOptions := []resourceSdk.UserTraitOption{
		resourceSdk.WithEmail(user.Email, true),
		resourceSdk.WithStatus(status),
		resourceSdk.WithUserProfile(profile),
		resourceSdk.WithUserLogin(user.LoginName),
	}

	userResource0, err := resourceSdk.NewUserResource(
		getDisplayName(user),
		userResourceType,
		user.Id,
		userTraitOptions,
		resourceSdk.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return userResource0, nil
}

// List method fetches a page of users and returns them as `v2.Resource` objects.
// It implements the `List` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method calls the Percipio API to get a page of users, transforms each user into a resource, and returns the list along with a pagination token.
// Which enables the baton-sdk to paginate through all user resources in the upstream system.
// This implementation uses the `client.ParseUserPaginationToken` and `client.GetUserNextToken` functions to handle pagination logic.
func (o *userBuilder) List(
	ctx context.Context,
	parentResourceID *v2.ResourceId,
	pToken *pagination.Token,
) (
	[]*v2.Resource,
	string,
	annotations.Annotations,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Starting Users List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	offset, limit, err := client.ParseUserPaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	users, total, ratelimitData, err := o.client.GetUsers(ctx, offset, limit)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	for _, user := range users {
		userResource0, err := userResource(user, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, userResource0)
	}

	nextToken := client.GetUserNextToken(ctx, offset, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements method is a placeholder as users do not have entitlements themselves.
// It implements the `Entitlements` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method returns an empty slice because users are principals, not resources with entitlements.
// Which is a required part of the `ResourceSyncer` interface.
// This implementation correctly returns no entitlements for user resources.
func (o *userBuilder) Entitlements(
	_ context.Context,
	_ *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

// Grants method is a placeholder as users do not have grants to other principals.
// It implements the `Grants` method required by the `connectorbuilder.Dedupe` interface.
// The method returns an empty slice because user resources do not grant access in this model.
// Which is a required part of the `ResourceSyncer` interface.
// This implementation correctly returns no grants for user resources.
func (o *userBuilder) Grants(
	_ context.Context,
	_ *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

// newUserBuilder function creates a new `userBuilder`.
// It implements the constructor for the user resource syncer.
// The function initializes a `userBuilder` with an API client and the user resource type.
// Which provides a configured syncer ready to be used by the main connector.
// This implementation sets up the builder with its required dependencies.
func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client:       client,
		resourceType: userResourceType,
	}
}
