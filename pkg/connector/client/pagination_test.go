package client

import (
	"encoding/json"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePaginationToken_Courses(t *testing.T) {
	tests := []struct {
		name                string
		token               *pagination.Token
		expectedOffset      int
		expectedLimit       int
		expectedPagingReqID string
		expectError         bool
	}{
		{
			name:                "nil token uses defaults",
			token:               nil,
			expectedOffset:      0,
			expectedLimit:       PageSizeDefault,
			expectedPagingReqID: "",
			expectError:         false,
		},
		{
			name: "empty token uses defaults",
			token: &pagination.Token{
				Token: "",
				Size:  0,
			},
			expectedOffset:      0,
			expectedLimit:       PageSizeDefault,
			expectedPagingReqID: "",
			expectError:         false,
		},
		{
			name: "custom size overrides default",
			token: &pagination.Token{
				Token: "",
				Size:  500,
			},
			expectedOffset:      0,
			expectedLimit:       500,
			expectedPagingReqID: "",
			expectError:         false,
		},
		{
			name: "valid course pagination token",
			token: &pagination.Token{
				Token: `{"pagingRequestId":"test-uuid-123","offset":1000}`,
				Size:  1000,
			},
			expectedOffset:      1000,
			expectedLimit:       1000,
			expectedPagingReqID: "test-uuid-123",
			expectError:         false,
		},
		{
			name: "course token without pagingRequestId",
			token: &pagination.Token{
				Token: `{"pagingRequestId":"","offset":2000}`,
				Size:  1000,
			},
			expectedOffset:      2000,
			expectedLimit:       1000,
			expectedPagingReqID: "",
			expectError:         false,
		},
		{
			name: "invalid JSON token",
			token: &pagination.Token{
				Token: `{invalid json}`,
				Size:  1000,
			},
			expectedOffset:      0,
			expectedLimit:       0,
			expectedPagingReqID: "",
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit, pagingReqID, err := ParsePaginationToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOffset, offset)
			assert.Equal(t, tt.expectedLimit, limit)
			assert.Equal(t, tt.expectedPagingReqID, pagingReqID)
		})
	}
}

func TestParseSimplePaginationToken_Users(t *testing.T) {
	tests := []struct {
		name           string
		token          *pagination.Token
		expectedOffset int
		expectedLimit  int
		expectError    bool
	}{
		{
			name:           "nil token uses defaults",
			token:          nil,
			expectedOffset: 0,
			expectedLimit:  PageSizeDefault,
			expectError:    false,
		},
		{
			name: "empty token uses defaults",
			token: &pagination.Token{
				Token: "",
				Size:  0,
			},
			expectedOffset: 0,
			expectedLimit:  PageSizeDefault,
			expectError:    false,
		},
		{
			name: "custom size overrides default",
			token: &pagination.Token{
				Token: "",
				Size:  250,
			},
			expectedOffset: 0,
			expectedLimit:  250,
			expectError:    false,
		},
		{
			name: "valid simple pagination token",
			token: &pagination.Token{
				Token: `{"offset":500}`,
				Size:  1000,
			},
			expectedOffset: 500,
			expectedLimit:  1000,
			expectError:    false,
		},
		{
			name: "large offset value",
			token: &pagination.Token{
				Token: `{"offset":50000}`,
				Size:  1000,
			},
			expectedOffset: 50000,
			expectedLimit:  1000,
			expectError:    false,
		},
		{
			name: "invalid JSON token",
			token: &pagination.Token{
				Token: `{invalid json}`,
				Size:  1000,
			},
			expectedOffset: 0,
			expectedLimit:  0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit, err := ParseSimplePaginationToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOffset, offset)
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

func TestGetNextToken_Courses(t *testing.T) {
	tests := []struct {
		name            string
		offset          int
		limit           int
		total           int
		pagingRequestId string
		expectedToken   string
		expectEmpty     bool
	}{
		{
			name:            "no more pages",
			offset:          1000,
			limit:           1000,
			total:           1500,
			pagingRequestId: "test-uuid",
			expectedToken:   "",
			expectEmpty:     true,
		},
		{
			name:            "exact boundary no more pages",
			offset:          1000,
			limit:           1000,
			total:           2000,
			pagingRequestId: "test-uuid",
			expectedToken:   "",
			expectEmpty:     true,
		},
		{
			name:            "has next page",
			offset:          0,
			limit:           1000,
			total:           2500,
			pagingRequestId: "test-uuid-123",
			expectedToken:   `{"pagingRequestId":"test-uuid-123","offset":1000}`,
			expectEmpty:     false,
		},
		{
			name:            "has next page without pagingRequestId",
			offset:          1000,
			limit:           1000,
			total:           2500,
			pagingRequestId: "",
			expectedToken:   `{"pagingRequestId":"","offset":2000}`,
			expectEmpty:     false,
		},
		{
			name:            "safety limit reached - page 100",
			offset:          99000, // Page 100 (99000/1000 + 1 = 100)
			limit:           1000,
			total:           200000,
			pagingRequestId: "test-uuid",
			expectedToken:   "",
			expectEmpty:     true,
		},
		{
			name:            "safety limit not reached - page 99",
			offset:          98000, // Page 99 (98000/1000 + 1 = 99)
			limit:           1000,
			total:           200000,
			pagingRequestId: "test-uuid",
			expectedToken:   `{"pagingRequestId":"test-uuid","offset":99000}`,
			expectEmpty:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := GetNextToken(tt.offset, tt.limit, tt.total, tt.pagingRequestId)

			if tt.expectEmpty {
				assert.Empty(t, token, "Expected empty token")
				return
			}

			assert.NotEmpty(t, token, "Expected non-empty token")
			assert.Equal(t, tt.expectedToken, token)

			// Verify the token can be parsed back
			var parsed Pagination
			err := json.Unmarshal([]byte(token), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.offset+tt.limit, parsed.Offset)
			assert.Equal(t, tt.pagingRequestId, parsed.PagingRequestId)
		})
	}
}

func TestGetSimpleNextToken_Users(t *testing.T) {
	tests := []struct {
		name          string
		offset        int
		limit         int
		total         int
		expectedToken string
		expectEmpty   bool
	}{
		{
			name:          "no more pages",
			offset:        500,
			limit:         1000,
			total:         1200,
			expectedToken: "",
			expectEmpty:   true,
		},
		{
			name:          "exact boundary no more pages",
			offset:        1000,
			limit:         1000,
			total:         2000,
			expectedToken: "",
			expectEmpty:   true,
		},
		{
			name:          "has next page",
			offset:        0,
			limit:         1000,
			total:         2500,
			expectedToken: `{"offset":1000}`,
			expectEmpty:   false,
		},
		{
			name:          "has next page - middle pagination",
			offset:        1000,
			limit:         1000,
			total:         3000,
			expectedToken: `{"offset":2000}`,
			expectEmpty:   false,
		},
		{
			name:          "safety limit reached - page 100",
			offset:        99000, // Page 100
			limit:         1000,
			total:         200000,
			expectedToken: "",
			expectEmpty:   true,
		},
		{
			name:          "safety limit not reached - page 99",
			offset:        98000, // Page 99
			limit:         1000,
			total:         200000,
			expectedToken: `{"offset":99000}`,
			expectEmpty:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := GetSimpleNextToken(tt.offset, tt.limit, tt.total)

			if tt.expectEmpty {
				assert.Empty(t, token, "Expected empty token")
				return
			}

			assert.NotEmpty(t, token, "Expected non-empty token")
			assert.Equal(t, tt.expectedToken, token)

			// Verify the token can be parsed back
			var parsed SimplePagination
			err := json.Unmarshal([]byte(token), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.offset+tt.limit, parsed.Offset)
		})
	}
}

func TestPaginationSafetyLimits(t *testing.T) {
	t.Run("MaxPagesPerSync constant", func(t *testing.T) {
		// Verify the safety limit is reasonable
		assert.Equal(t, 100, MaxPagesPerSync)

		// With default page size, this allows 100,000 items
		maxItems := MaxPagesPerSync * PageSizeDefault
		assert.Equal(t, 100000, maxItems)
	})

	t.Run("courses pagination stops at safety limit", func(t *testing.T) {
		// Test at exactly the limit
		offset := (MaxPagesPerSync - 1) * 1000 // Page 100
		token := GetNextToken(offset, 1000, 1000000, "test-uuid")
		assert.Empty(t, token, "Should stop at safety limit")
	})

	t.Run("users pagination stops at safety limit", func(t *testing.T) {
		// Test at exactly the limit
		offset := (MaxPagesPerSync - 1) * 1000 // Page 100
		token := GetSimpleNextToken(offset, 1000, 1000000)
		assert.Empty(t, token, "Should stop at safety limit")
	})
}

func TestPaginationTokenRoundTrip(t *testing.T) {
	t.Run("courses pagination round trip", func(t *testing.T) {
		// Create a token
		originalToken := GetNextToken(1000, 1000, 5000, "test-uuid-123")
		require.NotEmpty(t, originalToken)

		// Parse it back
		pToken := &pagination.Token{
			Token: originalToken,
			Size:  1000,
		}
		offset, limit, pagingReqID, err := ParsePaginationToken(pToken)
		require.NoError(t, err)

		// Verify values
		assert.Equal(t, 2000, offset) // 1000 + 1000
		assert.Equal(t, 1000, limit)
		assert.Equal(t, "test-uuid-123", pagingReqID)
	})

	t.Run("users pagination round trip", func(t *testing.T) {
		// Create a token
		originalToken := GetSimpleNextToken(500, 1000, 3000)
		require.NotEmpty(t, originalToken)

		// Parse it back
		pToken := &pagination.Token{
			Token: originalToken,
			Size:  1000,
		}
		offset, limit, err := ParseSimplePaginationToken(pToken)
		require.NoError(t, err)

		// Verify values
		assert.Equal(t, 1500, offset) // 500 + 1000
		assert.Equal(t, 1000, limit)
	})
}

func TestPaginationEdgeCases(t *testing.T) {
	t.Run("zero total items", func(t *testing.T) {
		token := GetNextToken(0, 1000, 0, "test-uuid")
		assert.Empty(t, token)

		simpleToken := GetSimpleNextToken(0, 1000, 0)
		assert.Empty(t, simpleToken)
	})

	t.Run("single item", func(t *testing.T) {
		token := GetNextToken(0, 1000, 1, "test-uuid")
		assert.Empty(t, token)

		simpleToken := GetSimpleNextToken(0, 1000, 1)
		assert.Empty(t, simpleToken)
	})

	t.Run("small page size", func(t *testing.T) {
		// Test with page size 1
		token := GetNextToken(0, 1, 5, "test-uuid")
		assert.Equal(t, `{"pagingRequestId":"test-uuid","offset":1}`, token)

		simpleToken := GetSimpleNextToken(0, 1, 5)
		assert.Equal(t, `{"offset":1}`, simpleToken)
	})

	t.Run("offset equals total", func(t *testing.T) {
		token := GetNextToken(1000, 1000, 1000, "test-uuid")
		assert.Empty(t, token)

		simpleToken := GetSimpleNextToken(1000, 1000, 1000)
		assert.Empty(t, simpleToken)
	})
}
