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
		require.Len(t, resources, 3)
		require.NotEmpty(t, resources[0].Id)
	})

	t.Run("should get limited courses using the search endpoint", func(t *testing.T) {
		limitCourseID := "1a3a3f54-b601-4d45-a234-038c980ee20f"
		limitCourses := mapset.NewSet(limitCourseID)
		c := newCourseBuilder(percipioClient, limitCourses)

		resources, nextToken, listAnnotations, err := c.List(ctx, nil, &pagination.Token{})
		require.Nil(t, err)
		test.AssertNoRatelimitAnnotations(t, listAnnotations)
		require.Empty(t, nextToken, "next token should be empty when searching by id")

		require.NotNil(t, resources)
		require.Len(t, resources, 1)

		assert.Equal(t, limitCourseID, resources[0].Id.Resource)
		assert.Equal(t, "Case Studies: Successful Data Privacy Implementations", resources[0].DisplayName)
	})

	t.Run("should list grants", func(t *testing.T) {
		c := newCourseBuilder(percipioClient, nil)
		course, _ := courseResource(ctx, client.Course{
			Id: "00000000-0000-0000-0000-000000000000",
			ContentType: client.ContentType{
				PercipioType: "COURSE",
				Category:     "COURSE",
				DisplayLabel: "Course",
			},
		}, nil)
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
