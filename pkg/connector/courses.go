package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	mapset "github.com/deckarep/golang-set/v2"
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
	limitCourses mapset.Set[string]
}

func (o *courseBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return o.resourceType
}

func courseResource(course client.Course, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	courseName := ""

	for _, metadata := range course.LocalizedMetadata {
		if metadata.Title == "" {
			continue
		}

		// American is the best language. Default to it.
		if metadata.LocaleCode == "en-US" {
			courseName = metadata.Title
			break
		}

		if courseName == "" {
			courseName = metadata.Title
		}
	}

	if courseName == "" {
		courseName = course.Code
	}
	if courseName == "" {
		courseName = course.Id
	}

	resource, err := resourceSdk.NewResource(
		courseName,
		courseResourceType,
		course.Id,
		resourceSdk.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
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

	courses, pagingRequestId, total, ratelimitData, err := o.client.GetCourses(
		ctx,
		offset,
		limit,
		pagingRequestId,
	)
	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	for _, course := range courses {
		if course.Lifecycle.Status == "INACTIVE" {
			continue
		}
		if o.limitCourses != nil && !o.limitCourses.Contains(course.Id) {
			continue
		}
		resource, err := courseResource(course, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		outputResources = append(outputResources, resource)
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

// Grants we have to do a pretty complicated set of maneuvers here to fetch
// grants. First, we need to POST a request to the "generate report" endpoint,
// which returns a UUID that we can use to interpolate a URL where the report
// will appear. From there we have to _poll_ that endpoint until it states that
// the report is ready. Finally, we need to store the data (which can be on the
// order of 1 GB) in memory so that we can find grants for a given resource.
func (o *courseBuilder) Grants(
	ctx context.Context,
	resource *v2.Resource,
	_ *pagination.Token,
) (
	[]*v2.Grant,
	string,
	annotations.Annotations,
	error,
) {
	var outputAnnotations annotations.Annotations
	if o.client.ReportStatus.Status == "" {
		ratelimitData, err := o.client.GenerateLearningActivityReport(ctx)
		outputAnnotations.WithRateLimiting(ratelimitData)
		if err != nil {
			return nil, "", outputAnnotations, err
		}
	}

	if o.client.ReportStatus.Status == "PENDING" || o.client.ReportStatus.Status == "IN_PROGRESS" {
		ratelimitData, err := o.client.GetLearningActivityReport(ctx)
		outputAnnotations.WithRateLimiting(ratelimitData)
		if err != nil {
			return nil, "", outputAnnotations, err
		}
	}

	statusesMap := o.client.StatusesStore.Get(resource.Id.Resource)

	grants := make([]*v2.Grant, 0)
	for userId, status := range statusesMap {
		principalId, err := resourceSdk.NewResourceID(userResourceType, userId)
		if err != nil {
			return nil, "", outputAnnotations, err
		}
		nextGrant := grant.NewGrant(resource, status, principalId)
		grants = append(grants, nextGrant)
	}

	return grants, "", outputAnnotations, nil
}

func newCourseBuilder(client *client.Client, limitCourses mapset.Set[string]) *courseBuilder {
	return &courseBuilder{
		client:       client,
		resourceType: courseResourceType,
		limitCourses: limitCourses,
	}
}
