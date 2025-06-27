package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// UserPagination struct holds the state for standard offset-based pagination.
// It is used by the user management API endpoints.
// It holds the `Offset` field, representing the starting point for the next page of results.
// This structure organizes the pagination token for APIs that use a simple offset and limit system.
// Instances are serialized into a JSON string to form the `pToken.Token` for the next page request.
type UserPagination struct {
	Offset int `json:"offset"`
}

// ContentPagination struct holds the state for the non-standard, stateful content discovery pagination.
// It is required by the `/catalog-content` endpoint.
// It holds fields such as `Offset`, `PagingRequestId`, and `FinalOffset` to manage the complex pagination flow.
// This structure organizes the pagination token for the content API, which requires a unique ID for subsequent requests.
// Instances are serialized into a JSON string to maintain state between paginated calls.
type ContentPagination struct {
	Offset          int    `json:"offset"`
	PagingRequestId string `json:"pagingRequestId"`
	FinalOffset     int    `json:"finalOffset"`
}

// ParseUserPaginationToken function decodes the pagination token for the user management API.
// It implements the token parsing required by any user-related resource syncer.
// The function deserializes the JSON pagination token from the SDK's `pToken` and extracts the next offset.
// Which allows the connector to resume pagination from where the previous API call left off.
// This implementation is aligned with standard baton-sdk pagination patterns.
func ParseUserPaginationToken(pToken *pagination.Token) (int, int, error) {
	logger := zap.L()

	var (
		limit  = PageSizeDefault
		offset = 0
	)

	if pToken == nil {
		logger.Debug("ParseUserPaginationToken: nil token, using defaults",
			zap.Int("defaultLimit", limit),
			zap.Int("defaultOffset", offset),
		)
		return offset, limit, nil
	}

	logger.Debug("ParseUserPaginationToken called",
		zap.String("token", pToken.Token),
		zap.Int("size", pToken.Size),
	)

	if pToken.Size > 0 {
		limit = pToken.Size
	}

	if pToken.Token != "" {
		var parsed UserPagination
		err := json.Unmarshal([]byte(pToken.Token), &parsed)
		if err != nil {
			logger.Error("ParseUserPaginationToken: failed to unmarshal token",
				zap.String("token", pToken.Token),
				zap.Error(err),
			)
			return 0, 0, err
		}
		offset = parsed.Offset
	}

	logger.Debug("ParseUserPaginationToken result",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	return offset, limit, nil
}

// GetUserNextToken function generates the next pagination token for the user management API.
// It implements the token generation for standard offset-based pagination.
// The function calculates the next offset and serializes it into a `UserPagination` JSON string.
// Which creates the token needed by the baton-sdk to request the subsequent page of users.
// This implementation returns an empty string when the last page is reached, signaling the end of pagination.
func GetUserNextToken(ctx context.Context, offset, limit, total int) string {
	l := ctxzap.Extract(ctx)
	nextOffset := offset + limit

	l.Debug("GetUserNextToken called",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.Int("total", total),
		zap.Int("nextOffset", nextOffset),
	)

	if nextOffset >= total {
		l.Debug("GetUserNextToken: pagination complete",
			zap.Int("nextOffset", nextOffset),
			zap.Int("total", total),
		)
		return ""
	}

	bytes, err := json.Marshal(UserPagination{Offset: nextOffset})
	if err != nil {
		l.Error("GetUserNextToken: failed to marshal pagination token",
			zap.Int("nextOffset", nextOffset),
			zap.Error(err),
		)
		return ""
	}

	nextToken := string(bytes)
	l.Debug("GetUserNextToken: token generated",
		zap.String("nextToken", nextToken),
	)

	return nextToken
}

// ParseContentPaginationToken function decodes the pagination token for the content discovery API.
// It implements the token parsing for Percipio's non-standard, stateful content pagination.
// The function deserializes the JSON pagination token and extracts the `Offset`, `PagingRequestId`, and `FinalOffset`.
// Which allows the connector to maintain the complex state required between calls to the content endpoint.
// This implementation is specific to the unique requirements of the `/catalog-content` API.
func ParseContentPaginationToken(ctx context.Context, pToken *pagination.Token) (int, string, int, error) {
	l := ctxzap.Extract(ctx)
	var (
		offset          = 0
		pagingRequestId = ""
		finalOffset     = 0
	)

	if pToken != nil && pToken.Token != "" {
		var parsed ContentPagination
		err := json.Unmarshal([]byte(pToken.Token), &parsed)
		if err != nil {
			l.Error("ParseContentPaginationToken: failed to unmarshal token",
				zap.String("token", pToken.Token),
				zap.Error(err),
			)
			return 0, "", 0, err
		}
		offset = parsed.Offset
		pagingRequestId = parsed.PagingRequestId
		finalOffset = parsed.FinalOffset
	}

	l.Debug("ParseContentPaginationToken result",
		zap.Int("offset", offset),
		zap.String("pagingRequestId", pagingRequestId),
		zap.Int("finalOffset", finalOffset),
	)

	return offset, pagingRequestId, finalOffset, nil
}

// GetContentNextToken function generates the next pagination token for the content discovery API.
// It implements the token generation for Percipio's non-standard, stateful content pagination.
// The function calculates the next offset and serializes it along with the required `PagingRequestId` and `FinalOffset` into a JSON string.
// Which creates the stateful token needed to request the subsequent page of content.
// This implementation returns an empty string when the final offset is reached, signaling the end of pagination.
func GetContentNextToken(ctx context.Context, currentOffset, limit, finalOffset int, pagingRequestId string) string {
	l := ctxzap.Extract(ctx)
	nextOffset := currentOffset + limit

	if nextOffset > finalOffset {
		l.Debug("GetContentNextToken: pagination complete",
			zap.Int("currentOffset", currentOffset),
			zap.Int("finalOffset", finalOffset),
		)
		return ""
	}

	bytes, err := json.Marshal(ContentPagination{
		Offset:          nextOffset,
		PagingRequestId: pagingRequestId,
		FinalOffset:     finalOffset,
	})
	if err != nil {
		l.Error("GetContentNextToken: failed to marshal pagination token",
			zap.Int("currentOffset", currentOffset),
			zap.Error(err),
		)
		return ""
	}

	l.Debug("GetContentNextToken: token generated",
		zap.String("nextToken", string(bytes)),
	)

	return string(bytes)
}

// ParseLinkHeader function extracts the final offset from a `Link` HTTP header.
// It implements the parsing of the `rel="last"` URL, which is a specific requirement of the content discovery API's first response.
// The function uses a regular expression to find the `rel="last"` URL, parses it, and extracts the `offset` query parameter.
// Which is the only mechanism the API provides to determine the total number of content items for pagination.
// This implementation is a crucial helper for initiating the stateful content pagination flow.
func ParseLinkHeader(ctx context.Context, linkHeader string) (int, error) {
	l := ctxzap.Extract(ctx)

	l.Info("Content pagination: parsing link header for final offset",
		zap.String("linkHeader", linkHeader),
	)

	lastLinkRegex := regexp.MustCompile(`<([^>]+)>;\s*[^,]*rel="last"`)
	matches := lastLinkRegex.FindStringSubmatch(linkHeader)

	if len(matches) < 2 {
		l.Error("ParseLinkHeader: no rel=last found in link header")
		return 0, fmt.Errorf("no rel=last found in link header")
	}

	lastURL := matches[1]
	l.Debug("ParseLinkHeader: found rel=last URL", zap.String("lastURL", lastURL))

	parsedURL, err := url.Parse(lastURL)
	if err != nil {
		l.Error("ParseLinkHeader: failed to parse last URL", zap.String("lastURL", lastURL), zap.Error(err))
		return 0, fmt.Errorf("failed to parse last URL: %w", err)
	}

	offsetStr := parsedURL.Query().Get("offset")
	if offsetStr == "" {
		l.Error("ParseLinkHeader: no offset parameter found in last URL", zap.String("lastURL", lastURL))
		return 0, fmt.Errorf("no offset parameter found in last URL")
	}

	finalOffset, err := strconv.Atoi(offsetStr)
	if err != nil {
		l.Error("ParseLinkHeader: failed to parse offset value", zap.String("offsetStr", offsetStr), zap.Error(err))
		return 0, fmt.Errorf("failed to parse offset value: %w", err)
	}

	l.Info("Content pagination: extracted final offset from link header",
		zap.String("linkHeader", linkHeader),
		zap.Int("finalOffset", finalOffset),
		zap.String("explanation", "Pagination will stop when currentOffset >= finalOffset"),
	)

	return finalOffset, nil
}
