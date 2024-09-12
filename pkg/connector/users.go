package connector

import (
	"context"
	"strconv"
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

type userBuilder struct {
	client       *client.Client
	resourceType *v2.ResourceType
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return o.resourceType
}

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

// Create a new connector resource for a Percipio user.
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

	// TODO(marcos): check that "isActive" means that the user is active.
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

// parsePaginationToken - takes as pagination token and returns offset and limit in that order.
func parsePaginationToken(pToken *pagination.Token) (int, int, error) {
	var limit = client.PageSizeDefault
	var offset = 0

	if pToken != nil {
		if pToken.Size > 0 {
			limit = pToken.Size
		}

		if pToken.Token != "" {
			parsedOffset, err := strconv.Atoi(pToken.Token)
			if err != nil {
				return 0, 0, err
			}
			offset = parsedOffset
		}
	}
	return offset, limit, nil
}

func getNextToken(offset int, limit int, total int) string {
	nextOffset := offset + limit
	if nextOffset >= total {
		return ""
	}
	return strconv.Itoa(nextOffset)
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
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

	offset, limit, err := parsePaginationToken(pToken)
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

	nextToken := getNextToken(offset, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
func (o *userBuilder) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *userBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	pToken *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	return nil, "", nil, nil
}

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client:       client,
		resourceType: userResourceType,
	}
}
