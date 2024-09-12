package connector

import (
	"context"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
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

	offset, limit, err := parsePaginationToken(pToken)
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

	nextToken := getNextToken(offset, limit, total)

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements always returns an empty slice for users.
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
	return nil, "", nil, nil
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
