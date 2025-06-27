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
	completedEntitlement  = "completed"
	inProgressEntitlement = "in_progress"
)

// courseBuilder struct is responsible for syncing course resources and their associated grants.
// It is used by the connector to fetch and process all course and assessment data from the Percipio API.
// It holds a reference to the API client, the course resource type descriptor, and a set of courses to limit the sync.
// This structure organizes the context needed for all course-related synchronization operations.
// Instances are created by the `newCourseBuilder` function.
type courseBuilder struct {
	client       *client.Client
	resourceType *v2.ResourceType
	limitCourses mapset.Set[string]
}

// ResourceType method returns the resource type descriptor for courses.
// It implements the `ResourceType` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method returns the static `courseResourceType` object defined for this connector.
// Which informs the baton-sdk about the type of resource this syncer is responsible for.
// This implementation returns a pre-defined object.
func (o *courseBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return o.resourceType
}

// courseResource function creates a new `v2.Resource` from a Percipio course object.
// It implements the mapping from the provider's content data model to the baton-sdk's resource model.
// The function filters out inactive or non-course/assessment content and constructs a display name before creating the resource.
// Which is the core transformation for converting raw course data into a standardized resource object that Baton can process.
// This implementation returns `nil` for any content that should be skipped, which is handled by the caller.
func courseResource(ctx context.Context, course client.Course, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	l := ctxzap.Extract(ctx)
	if course.Lifecycle.Status == "INACTIVE" {
		l.Debug("Skipping inactive course", zap.String("courseId", course.Id))
		return nil, nil
	}
	if course.ContentType.PercipioType != "COURSE" && course.ContentType.PercipioType != "ASSESSMENT" {
		l.Debug("Skipping non-course content", zap.String("courseId", course.Id), zap.String("contentType", course.ContentType.PercipioType))
		return nil, nil
	}

	resourceOpts := []resourceSdk.ResourceOption{
		resourceSdk.WithParentResourceID(parentResourceID),
	}

	courseName := ""
	for _, metadata := range course.LocalizedMetadata {
		if metadata.Title == "" {
			continue
		}

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
		resourceOpts...,
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

// List method fetches a page of courses and returns them as `v2.Resource` objects.
// It implements the `List` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method calls the Percipio API to get a page of content, transforms each item into a resource, and returns the list along with a pagination token.
// Which enables the baton-sdk to paginate through all course and assessment resources in the upstream system.
// This implementation uses the `client.ParseContentPaginationToken` and `client.GetContentNextToken` functions to handle Percipio's non-standard pagination logic.
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
	l := ctxzap.Extract(ctx)
	l.Debug("Starting Courses List", zap.String("token", pToken.Token))

	outputResources := make([]*v2.Resource, 0)
	var outputAnnotations annotations.Annotations

	// If limitCourses is set, we use the search endpoint instead of paginating
	if o.limitCourses != nil && o.limitCourses.Cardinality() > 0 {
		courseIDs := o.limitCourses.ToSlice()
		for _, courseID := range courseIDs {
			courses, ratelimitData, err := o.client.SearchContentByID(ctx, courseID)
			outputAnnotations.WithRateLimiting(ratelimitData)
			if err != nil {
				l.Warn("failed to find course by id", zap.Error(err), zap.String("courseID", courseID))
				continue
			}

			// The search endpoint can return multiple results, we need to find the exact match
			for _, course := range courses {
				if course.Id == courseID {
					resource, err := courseResource(ctx, course, parentResourceID)
					if err != nil {
						return nil, "", nil, err
					}
					if resource == nil {
						continue
					}
					outputResources = append(outputResources, resource)
				}
			}
		}

		return outputResources, "", outputAnnotations, nil
	}

	offset, pagingRequestId, finalOffset, err := client.ParseContentPaginationToken(ctx, pToken)
	if err != nil {
		return nil, "", nil, err
	}

	courses, newPagingRequestId, returnedFinalOffset, ratelimitData, err := o.client.GetCourses(
		ctx,
		offset,
		1000,
		pagingRequestId,
	)

	if finalOffset == 0 && returnedFinalOffset > 0 {
		finalOffset = returnedFinalOffset
	}

	hasMore := offset <= finalOffset
	var nextOffset int
	if hasMore {
		nextOffset = offset + 1000
	}

	l.Info("Content pagination progress",
		zap.Int("currentOffset", offset),
		zap.Int("finalOffset", finalOffset),
		zap.Bool("hasMore", hasMore),
		zap.Int("nextOffset", nextOffset),
	)

	outputAnnotations.WithRateLimiting(ratelimitData)
	if err != nil {
		return nil, "", outputAnnotations, err
	}
	for _, course := range courses {
		if o.limitCourses != nil && !o.limitCourses.Contains(course.Id) {
			continue
		}
		resource, err := courseResource(ctx, course, parentResourceID)
		if err != nil {
			return nil, "", nil, err
		}
		if resource == nil {
			continue
		}
		outputResources = append(outputResources, resource)
	}

	nextToken := client.GetContentNextToken(ctx, offset, 1000, finalOffset, newPagingRequestId)

	if nextToken == "" {
		l.Info("Content pagination complete",
			zap.Int("finalOffset", finalOffset),
			zap.String("explanation", "Reached final offset, pagination stopped"),
		)
	}

	return outputResources, nextToken, outputAnnotations, nil
}

// Entitlements method returns the entitlements for a course resource.
// It implements the `Entitlements` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method defines the 'assigned', 'completed', and 'in_progress' entitlements for a given course.
// Which allows Baton to model the different types of relationships a user can have with a course.
// This implementation returns a static list of three assignment entitlements.
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

// Grants method fetches and returns the grants for a course resource.
// It implements the `Grants` method required by the `connectorbuilder.ResourceSyncer` interface.
// The method orchestrates a multi-step, asynchronous report generation process: it first requests a report,
// then polls for its completion, and finally processes the report data from an in-memory cache to create grants.
// Which is the only mechanism for determining user course entitlements in the Percipio API.
// This implementation relies on the client's `StatusesStore` to retrieve the report data fetched by the report syncer.
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

// newCourseBuilder function creates a new `courseBuilder`.
// It implements the constructor for the course resource syncer.
// The function initializes a `courseBuilder` with an API client, the course resource type, and a set of courses to limit the sync.
// Which provides a configured syncer ready to be used by the main connector.
// This implementation sets up the builder with its required dependencies.
func newCourseBuilder(client *client.Client, limitCourses mapset.Set[string]) *courseBuilder {
	return &courseBuilder{
		client:       client,
		resourceType: courseResourceType,
		limitCourses: limitCourses,
	}
}
