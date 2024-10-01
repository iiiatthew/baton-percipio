package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	assignedEntitlement   = "assigned"
	completedEntitlement  = "completed"
	inProgressEntitlement = "in_progress"
)

type courseBuilder struct {
	client       *client.Client
	resourceType *v2.ResourceType
}

func (o *courseBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return o.resourceType
}

func courseResource(course client.Course, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	resource0, err := resourceSdk.NewResource(
		course.Code,
		userResourceType,
		course.Id,
		resourceSdk.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return resource0, nil
}

func (o *courseBuilder) List(
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
	logger.Debug("Starting Courses List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	offset, limit, pagingRequestId, err := client.ParsePaginationToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	courses, total, ratelimitData, err := o.client.GetCourses(ctx, offset, limit)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	for _, course := range courses {
		resource0, err := courseResource(course, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource0)
	}

	nextToken := client.GetNextToken(offset, limit, total, pagingRequestId)

	return outputResources, nextToken, outputAnnotations, nil
}

func (o *courseBuilder) Entitlements(
	_ context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Entitlement,
	string,
	annotations.Annotations,
	error,
) {
	return []*v2.Entitlement{
		entitlement.NewAssignmentEntitlement(
			resource,
			assignedEntitlement,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(fmt.Sprintf("Course %s %s", resource.DisplayName, assignedEntitlement)),
			entitlement.WithDescription(fmt.Sprintf("Assigned course %s in Percipio", resource.DisplayName)),
		),
		entitlement.NewAssignmentEntitlement(
			resource,
			completedEntitlement,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(fmt.Sprintf("Course %s %s", resource.DisplayName, completedEntitlement)),
			entitlement.WithDescription(fmt.Sprintf("Completed course %s in Percipio", resource.DisplayName)),
		),
		entitlement.NewAssignmentEntitlement(
			resource,
			inProgressEntitlement,
			entitlement.WithGrantableTo(userResourceType),
			entitlement.WithDisplayName(fmt.Sprintf("Course %s %s", resource.DisplayName, inProgressEntitlement)),
			entitlement.WithDescription(fmt.Sprintf("In progress course %s in Percipio", resource.DisplayName)),
		),
	}, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (o *courseBuilder) Grants(
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

func newCourseBuilder(client *client.Client) *courseBuilder {
	return &courseBuilder{
		client:       client,
		resourceType: courseResourceType,
	}
}
