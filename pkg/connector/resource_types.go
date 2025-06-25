package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

// annotationsForUserResourceType function creates the annotations for the user resource type.
// It implements the annotation logic required by the user resource type definition.
// The function creates a `SkipEntitlementsAndGrants` annotation to inform the SDK that users, as principals, do not have their own entitlements or grants.
// Which is a necessary configuration for principal resource types in the baton-sdk.
// This implementation returns a new `annotations.Annotations` object with the appropriate annotation added.
func annotationsForUserResourceType() annotations.Annotations {
	annos := annotations.Annotations{}
	annos.Update(&v2.SkipEntitlementsAndGrants{})
	return annos
}

// userResourceType is the resource type descriptor for users.
// It is used by the user resource syncer to define the user resource type.
// It holds the `Id`, `DisplayName`, `Traits`, and `Annotations` for the user resource type.
// This variable defines the schema for user resources in Baton.
// The instance is configured with the `TRAIT_USER` trait and annotations to skip grants and entitlements.
var userResourceType = &v2.ResourceType{
	Id:          "user",
	DisplayName: "User",
	Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
	Annotations: annotationsForUserResourceType(),
}

// courseResourceType is the resource type descriptor for courses.
// It is used by the course resource syncer to define the course resource type.
// It holds the `Id` and `DisplayName` for the course resource type.
// This variable defines the schema for course resources in Baton.
// The instance is configured with a simple ID and display name.
var courseResourceType = &v2.ResourceType{
	Id:          "course",
	DisplayName: "course",
}
