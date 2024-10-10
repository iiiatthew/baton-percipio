package connector

import (
	"context"
	"testing"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	"github.com/conductorone/baton-percipio/test"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoursesList(t *testing.T) {
	ctx := context.Background()
	server := test.FixturesServer()
	defer server.Close()

	percipioClient, err := client.New(
		ctx,
		server.URL,
		"mock",
		"token",
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("should get all courses with pagination", func(t *testing.T) {
		c := newCourseBuilder(percipioClient, nil)
		resources := make([]*v2.Resource, 0)
		pToken := pagination.Token{
			Token: "",
			Size:  1,
		}
		for {
			nextResources, nextToken, listAnnotations, err := c.List(ctx, nil, &pToken)
			resources = append(resources, nextResources...)

			require.Nil(t, err)
			test.AssertNoRatelimitAnnotations(t, listAnnotations)
			if nextToken == "" {
				break
			}

			pToken.Token = nextToken
		}

		require.NotNil(t, resources)
		require.Len(t, resources, 2)
		require.NotEmpty(t, resources[0].Id)
	})

	t.Run("should get limited courses", func(t *testing.T) {
		limitCourseID := "00000000-0000-0000-0000-000000000000"
		limitCourses := mapset.NewSet(limitCourseID)
		c := newCourseBuilder(percipioClient, limitCourses)
		resources := make([]*v2.Resource, 0)
		pToken := pagination.Token{
			Token: "",
			Size:  1,
		}
		for {
			nextResources, nextToken, listAnnotations, err := c.List(ctx, nil, &pToken)
			resources = append(resources, nextResources...)

			require.Nil(t, err)
			test.AssertNoRatelimitAnnotations(t, listAnnotations)
			if nextToken == "" {
				break
			}

			pToken.Token = nextToken
		}

		require.NotNil(t, resources)
		require.NotEmpty(t, resources)
		for _, resource := range resources {
			assert.Equal(t, limitCourseID, resource.Id.Resource, "All returned resources should have the limited course ID")
		}
	})

	t.Run("should list grants", func(t *testing.T) {
		c := newCourseBuilder(percipioClient, nil)
		course, _ := courseResource(client.Course{Id: "00000000-0000-0000-0000-000000000000"}, nil)
		grants := make([]*v2.Grant, 0)
		pToken := pagination.Token{
			Token: "",
			Size:  100,
		}
		for {
			nextGrants, nextToken, listAnnotations, err := c.Grants(ctx, course, &pToken)
			grants = append(grants, nextGrants...)

			require.Nil(t, err)
			test.AssertNoRatelimitAnnotations(t, listAnnotations)
			if nextToken == "" {
				break
			}
			pToken.Token = nextToken
		}
		require.Len(t, grants, 1)
	})
}
