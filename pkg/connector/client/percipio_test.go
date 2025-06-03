package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderParsing(t *testing.T) {
	t.Run("getTotalCount with valid header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set(HeaderNameTotalCount, "150")

		total, err := getTotalCount(resp)
		require.NoError(t, err)
		assert.Equal(t, 150, total)
	})

	t.Run("getTotalCount with missing header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}

		total, err := getTotalCount(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing x-total-count header")
		assert.Equal(t, 0, total)
	})

	t.Run("getTotalCount with invalid header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set(HeaderNameTotalCount, "not-a-number")

		total, err := getTotalCount(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid x-total-count header value")
		assert.Equal(t, 0, total)
	})

	t.Run("getTotalCount with suspiciously high value", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set(HeaderNameTotalCount, "2000000") // 2 million

		total, err := getTotalCount(resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "suspiciously high total count")
		assert.Equal(t, 0, total)
	})
}

func TestContentDiscoveryHeaderParsing(t *testing.T) {
	t.Run("getTotalCountForContentDiscovery with header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set(HeaderNameTotalCount, "500")

		total, err := getTotalCountForContentDiscovery(resp, 100, 0, 1000)
		require.NoError(t, err)
		assert.Equal(t, 500, total)
	})

	t.Run("getTotalCountForContentDiscovery missing header - last page", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		// No x-total-count header

		// If we got fewer courses than requested, we're at the end
		total, err := getTotalCountForContentDiscovery(resp, 750, 2000, 1000)
		require.NoError(t, err)
		assert.Equal(t, 2750, total) // offset + coursesReturned
	})

	t.Run("getTotalCountForContentDiscovery missing header - not last page", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		// No x-total-count header

		// If we got full page, estimate higher
		total, err := getTotalCountForContentDiscovery(resp, 1000, 1000, 1000)
		require.NoError(t, err)
		assert.Equal(t, 3000, total) // offset + coursesReturned + 1000
	})

	t.Run("getTotalCountForContentDiscovery with invalid header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set(HeaderNameTotalCount, "invalid")

		total, err := getTotalCountForContentDiscovery(resp, 100, 0, 1000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid x-total-count header value")
		assert.Equal(t, 0, total)
	})

	t.Run("getTotalCountForContentDiscovery with suspiciously high header", func(t *testing.T) {
		resp := &http.Response{
			Header: make(http.Header),
		}
		resp.Header.Set(HeaderNameTotalCount, "5000000") // 5 million

		total, err := getTotalCountForContentDiscovery(resp, 100, 0, 1000)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "suspiciously high total count")
		assert.Equal(t, 0, total)
	})
}

func TestConstants(t *testing.T) {
	t.Run("API paths are correctly formatted", func(t *testing.T) {
		assert.Equal(t, "/content-discovery/v2/organizations/%s/catalog-content", ApiPathCoursesList)
		assert.Equal(t, "/user-management/v1/organizations/%s/users", ApiPathUsersList)
		assert.Equal(t, "/reporting/v1/organizations/%s/report-requests/learning-activity", ApiPathLearningActivityReport)
		assert.Equal(t, "/reporting/v1/organizations/%s/report-requests/%s", ApiPathReport)
	})

	t.Run("headers are correctly named", func(t *testing.T) {
		assert.Equal(t, "x-paging-request-id", HeaderNamePagingRequestId)
		assert.Equal(t, "x-total-count", HeaderNameTotalCount)
	})

	t.Run("default values are reasonable", func(t *testing.T) {
		assert.Equal(t, 1000, PageSizeDefault)
		assert.Equal(t, "https://api.percipio.com", BaseApiUrl)
	})
}

func TestAPIPathFormatting(t *testing.T) {
	tests := []struct {
		name         string
		pathTemplate string
		orgId        string
		expected     string
	}{
		{
			name:         "courses path",
			pathTemplate: ApiPathCoursesList,
			orgId:        "test-org-123",
			expected:     "/content-discovery/v2/organizations/test-org-123/catalog-content",
		},
		{
			name:         "users path",
			pathTemplate: ApiPathUsersList,
			orgId:        "another-org-456",
			expected:     "/user-management/v1/organizations/another-org-456/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string formatting
			formatted := sprintf(tt.pathTemplate, tt.orgId)
			assert.Equal(t, tt.expected, formatted)
		})
	}
}

// Helper function for string formatting since we can't import fmt in this minimal test
func sprintf(format string, args ...interface{}) string {
	// Simple replacement for our test cases
	if len(args) == 1 {
		result := format
		for i := 0; i < len(result)-1; i++ {
			if result[i] == '%' && result[i+1] == 's' {
				return result[:i] + args[0].(string) + result[i+2:]
			}
		}
	}
	return format
}
