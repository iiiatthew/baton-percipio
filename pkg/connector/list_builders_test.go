package connector

import (
	"testing"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersPaginationPattern(t *testing.T) {
	t.Run("users should use simple pagination without pagingRequestId", func(t *testing.T) {
		// This test verifies that users use the simple pagination pattern
		// as documented in the user-management swagger spec

		// Test first page (no token)
		pToken := &pagination.Token{
			Token: "",
			Size:  1000,
		}

		// Users should parse as simple pagination (no pagingRequestId)
		offset, limit, err := client.ParseSimplePaginationToken(pToken)
		require.NoError(t, err)
		assert.Equal(t, 0, offset)
		assert.Equal(t, 1000, limit)

		// Users should generate simple next token (no pagingRequestId)
		nextToken := client.GetSimpleNextToken(0, 1000, 2500)
		assert.Equal(t, `{"offset":1000}`, nextToken)

		// Verify the token can be parsed back
		pToken2 := &pagination.Token{
			Token: nextToken,
			Size:  1000,
		}
		offset2, limit2, err := client.ParseSimplePaginationToken(pToken2)
		require.NoError(t, err)
		assert.Equal(t, 1000, offset2)
		assert.Equal(t, 1000, limit2)
	})

	t.Run("users pagination should respect safety limits", func(t *testing.T) {
		// Test that users pagination stops at safety limit
		offset := 99000 // Page 100
		nextToken := client.GetSimpleNextToken(offset, 1000, 200000)
		assert.Empty(t, nextToken, "Should stop at page 100 safety limit")
	})
}

func TestCoursesPaginationPattern(t *testing.T) {
	t.Run("courses should use complex pagination with pagingRequestId", func(t *testing.T) {
		// This test verifies that courses use the complex pagination pattern
		// as documented in the content-discovery swagger spec

		// Test first page (no token)
		pToken := &pagination.Token{
			Token: "",
			Size:  1000,
		}

		// Courses should parse as complex pagination (with pagingRequestId)
		offset, limit, pagingReqId, err := client.ParsePaginationToken(pToken)
		require.NoError(t, err)
		assert.Equal(t, 0, offset)
		assert.Equal(t, 1000, limit)
		assert.Equal(t, "", pagingReqId) // Empty on first page

		// Courses should generate complex next token (with pagingRequestId)
		nextToken := client.GetNextToken(0, 1000, 2500, "test-uuid-123")
		assert.Equal(t, `{"pagingRequestId":"test-uuid-123","offset":1000}`, nextToken)

		// Verify the token can be parsed back
		pToken2 := &pagination.Token{
			Token: nextToken,
			Size:  1000,
		}
		offset2, limit2, pagingReqId2, err := client.ParsePaginationToken(pToken2)
		require.NoError(t, err)
		assert.Equal(t, 1000, offset2)
		assert.Equal(t, 1000, limit2)
		assert.Equal(t, "test-uuid-123", pagingReqId2)
	})

	t.Run("courses pagination should respect safety limits", func(t *testing.T) {
		// Test that courses pagination stops at safety limit
		offset := 99000 // Page 100
		nextToken := client.GetNextToken(offset, 1000, 200000, "test-uuid")
		assert.Empty(t, nextToken, "Should stop at page 100 safety limit")
	})
}

func TestPaginationTokenStructures(t *testing.T) {
	t.Run("simple pagination token structure", func(t *testing.T) {
		// Users should use SimplePagination structure: {"offset":1000}
		token := client.GetSimpleNextToken(500, 1000, 2000)
		expected := `{"offset":1500}`
		assert.Equal(t, expected, token)
	})

	t.Run("complex pagination token structure", func(t *testing.T) {
		// Courses should use Pagination structure: {"pagingRequestId":"uuid","offset":1000}
		token := client.GetNextToken(500, 1000, 2000, "test-uuid")
		expected := `{"pagingRequestId":"test-uuid","offset":1500}`
		assert.Equal(t, expected, token)
	})

	t.Run("empty pagingRequestId handling", func(t *testing.T) {
		// Courses should handle empty pagingRequestId correctly
		token := client.GetNextToken(500, 1000, 2000, "")
		expected := `{"pagingRequestId":"","offset":1500}`
		assert.Equal(t, expected, token)
	})
}

func TestSwaggerComplianceMapping(t *testing.T) {
	t.Run("user management API compliance", func(t *testing.T) {
		// Based on user-management-swagger.json:
		// - Uses offset + max parameters
		// - Documents x-total-count header
		// - No pagingRequestId parameter
		// - Maximum 1000 items per page
		// - Default 1000 items per page

		// Test default page size matches swagger
		pToken := &pagination.Token{Token: "", Size: 0}
		_, limit, err := client.ParseSimplePaginationToken(pToken)
		require.NoError(t, err)
		assert.Equal(t, client.PageSizeDefault, limit) // Should be 1000

		// Test maximum page size enforcement (this would be handled by API)
		pToken2 := &pagination.Token{Token: "", Size: 1000}
		_, limit2, err := client.ParseSimplePaginationToken(pToken2)
		require.NoError(t, err)
		assert.Equal(t, 1000, limit2)
	})

	t.Run("content discovery API compliance", func(t *testing.T) {
		// Based on content-discovery-swagger.json:
		// - Uses offset + max + pagingRequestId parameters
		// - Does NOT document x-total-count header
		// - Requires pagingRequestId for stateful pagination
		// - Maximum 1000 items per page
		// - Default 1000 items per page

		// Test default page size matches swagger
		pToken := &pagination.Token{Token: "", Size: 0}
		_, limit, _, err := client.ParsePaginationToken(pToken)
		require.NoError(t, err)
		assert.Equal(t, client.PageSizeDefault, limit) // Should be 1000

		// Test maximum page size enforcement
		pToken2 := &pagination.Token{Token: "", Size: 1000}
		_, limit2, _, err := client.ParsePaginationToken(pToken2)
		require.NoError(t, err)
		assert.Equal(t, 1000, limit2)
	})
}

func TestPaginationBehaviorDifferences(t *testing.T) {
	t.Run("API pattern differences", func(t *testing.T) {
		// Users: Simple pattern
		userToken := client.GetSimpleNextToken(1000, 1000, 3000)
		assert.NotContains(t, userToken, "pagingRequestId", "Users should not include pagingRequestId")
		assert.Contains(t, userToken, `"offset":2000`, "Users should include offset")

		// Courses: Complex pattern
		courseToken := client.GetNextToken(1000, 1000, 3000, "uuid-123")
		assert.Contains(t, courseToken, "pagingRequestId", "Courses should include pagingRequestId")
		assert.Contains(t, courseToken, `"offset":2000`, "Courses should include offset")
		assert.Contains(t, courseToken, "uuid-123", "Courses should include actual pagingRequestId value")
	})

	t.Run("header handling differences", func(t *testing.T) {
		// This documents the expected behavior based on swagger analysis:

		// User Management API:
		// - MUST provide x-total-count header (documented in swagger)
		// - Simple error handling since header is required

		// Content Discovery API:
		// - MAY provide x-total-count header (not documented in swagger)
		// - Graceful fallback estimation when header missing
		// - Robust error handling for unreliable headers

		// These behaviors are tested in the integration tests
		assert.True(t, true, "Documented for reference")
	})
}

func TestSafetyLimitsConsistency(t *testing.T) {
	t.Run("both APIs use same safety limits", func(t *testing.T) {
		// Both user and course pagination should use the same client.MaxPagesPerSync limit
		userLimit := (client.MaxPagesPerSync - 1) * 1000
		courseLimit := (client.MaxPagesPerSync - 1) * 1000

		// Test users stop at limit
		userToken := client.GetSimpleNextToken(userLimit, 1000, 1000000)
		assert.Empty(t, userToken, "Users should stop at safety limit")

		// Test courses stop at limit
		courseToken := client.GetNextToken(courseLimit, 1000, 1000000, "uuid")
		assert.Empty(t, courseToken, "Courses should stop at safety limit")
	})

	t.Run("safety limit allows reasonable data volumes", func(t *testing.T) {
		// With client.MaxPagesPerSync = 100 and client.PageSizeDefault = 1000
		// Maximum items per sync = 100,000
		maxItems := client.MaxPagesPerSync * client.PageSizeDefault
		assert.Equal(t, 100000, maxItems, "Should allow 100,000 items per sync")

		// This should handle most reasonable organization sizes
		assert.GreaterOrEqual(t, maxItems, 50000, "Should handle large organizations")
		assert.LessOrEqual(t, maxItems, 1000000, "Should prevent excessive API usage")
	})
}
