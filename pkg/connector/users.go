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

	for _, attribute := range user.CustomAttributes {
		// TODO(marcos): Should I omit fields that are nullish? What about
		// overlapping field names?
		profile[attribute.Name] = attribute.Value
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

	// Use simple pagination (no pagingRequestId) for users endpoint
	offset, limit, err := client.ParseSimplePaginationToken(pToken)
	if err != nil {
		logger.Error("Failed to parse simple pagination token",
			zap.Error(err),
			zap.String("token", pToken.Token),
		)
		return nil, "", nil, err
	}

	logger.Info("Users List pagination info",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.String("token", pToken.Token),
	)

	users, total, ratelimitData, err := o.client.GetUsers(ctx, offset, limit)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		logger.Error("Failed to get users from API",
			zap.Error(err),
			zap.Int("offset", offset),
			zap.Int("limit", limit),
		)
		return nil, "", outputAnnotations, err
	}

	usersProcessed := 0
	for _, user := range users {
		userResource0, err := userResource(user, parentResourceID)
		if err != nil {
			logger.Error("Failed to create user resource",
				zap.Error(err),
				zap.String("userId", user.Id),
			)
			return nil, "", nil, err
		}
		outputResources = append(outputResources, userResource0)
		usersProcessed++
	}

	// Use simple pagination (no pagingRequestId) for users endpoint
	nextToken := client.GetSimpleNextToken(offset, limit, total)
	hasNextPage := nextToken != ""

	// Calculate progress metrics
	currentPage := (offset / limit) + 1
	totalPages := (total + limit - 1) / limit // Ceiling division
	progressPercent := float64(offset+len(users)) / float64(total) * 100

	logger.Info("Users List completed",
		zap.Int("usersFromAPI", len(users)),
		zap.Int("usersProcessed", usersProcessed),
		zap.Int("total", total),
		zap.Int("currentPage", currentPage),
		zap.Int("totalPages", totalPages),
		zap.Float64("progressPercent", progressPercent),
		zap.Bool("hasNextPage", hasNextPage),
		zap.String("nextToken", nextToken),
	)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
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

// Grants always returns an empty slice for users since they don't have any entitlements.
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

func newUserBuilder(client *client.Client) *userBuilder {
	return &userBuilder{
		client:       client,
		resourceType: userResourceType,
	}
}
