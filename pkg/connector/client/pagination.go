package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"

	"github.com/conductorone/baton-sdk/pkg/pagination"
	"go.uber.org/zap"
)

type UserPagination struct {
	Offset int `json:"offset"`
}

type ContentPagination struct {
	Offset          int    `json:"offset"`
	PagingRequestId string `json:"pagingRequestId"`
	FinalOffset     int    `json:"finalOffset"`
}

// ParseUserPaginationToken parses token for user management API.
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

// GetUserNextToken generates next token for user management API.
func GetUserNextToken(offset, limit, total int) string {
	logger := zap.L()
	nextOffset := offset + limit

	logger.Debug("GetUserNextToken called",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.Int("total", total),
		zap.Int("nextOffset", nextOffset),
	)

	if nextOffset >= total {
		logger.Debug("GetUserNextToken: pagination complete",
			zap.Int("nextOffset", nextOffset),
			zap.Int("total", total),
		)
		return ""
	}

	bytes, err := json.Marshal(UserPagination{Offset: nextOffset})
	if err != nil {
		logger.Error("GetUserNextToken: failed to marshal pagination token",
			zap.Int("nextOffset", nextOffset),
			zap.Error(err),
		)
		return ""
	}

	nextToken := string(bytes)
	logger.Debug("GetUserNextToken: token generated",
		zap.String("nextToken", nextToken),
	)

	return nextToken
}

// ParseContentPaginationToken parses token for content discovery API.
func ParseContentPaginationToken(pToken *pagination.Token) (int, string, int, error) {
	var (
		offset          = 0
		pagingRequestId = ""
		finalOffset     = 0
	)

	if pToken != nil && pToken.Token != "" {
		var parsed ContentPagination
		err := json.Unmarshal([]byte(pToken.Token), &parsed)
		if err != nil {
			return 0, "", 0, err
		}
		offset = parsed.Offset
		pagingRequestId = parsed.PagingRequestId
		finalOffset = parsed.FinalOffset
	}

	return offset, pagingRequestId, finalOffset, nil
}

// GetContentNextToken generates next token for content discovery API.
func GetContentNextToken(currentOffset, limit, finalOffset int, pagingRequestId string) string {
	logger := zap.L()
	nextOffset := currentOffset + limit

	if nextOffset > finalOffset {
		logger.Debug("GetContentNextToken: pagination complete",
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
		logger.Error("GetContentNextToken: failed to marshal pagination token",
			zap.Int("currentOffset", currentOffset),
			zap.Error(err),
		)
		return ""
	}

	return string(bytes)
}

// ParseLinkHeader extracts finalOffset from link header rel="last" section.
func ParseLinkHeader(linkHeader string) (int, error) {
	logger := zap.L()

	logger.Info("Content pagination: parsing link header for final offset",
		zap.String("linkHeader", linkHeader),
	)

	// Find the rel="last" section in the link header
	lastLinkRegex := regexp.MustCompile(`<([^>]+)>;\s*[^,]*rel="last"`)
	matches := lastLinkRegex.FindStringSubmatch(linkHeader)

	if len(matches) < 2 {
		logger.Error("ParseLinkHeader: no rel=last found in link header")
		return 0, fmt.Errorf("no rel=last found in link header")
	}

	lastURL := matches[1]
	logger.Debug("ParseLinkHeader: found rel=last URL", zap.String("lastURL", lastURL))

	// Parse the URL to extract offset parameter
	parsedURL, err := url.Parse(lastURL)
	if err != nil {
		logger.Error("ParseLinkHeader: failed to parse last URL", zap.String("lastURL", lastURL), zap.Error(err))
		return 0, fmt.Errorf("failed to parse last URL: %w", err)
	}

	offsetStr := parsedURL.Query().Get("offset")
	if offsetStr == "" {
		logger.Error("ParseLinkHeader: no offset parameter found in last URL", zap.String("lastURL", lastURL))
		return 0, fmt.Errorf("no offset parameter found in last URL")
	}

	finalOffset, err := strconv.Atoi(offsetStr)
	if err != nil {
		logger.Error("ParseLinkHeader: failed to parse offset value", zap.String("offsetStr", offsetStr), zap.Error(err))
		return 0, fmt.Errorf("failed to parse offset value: %w", err)
	}

	logger.Info("Content pagination: extracted final offset from link header",
		zap.String("linkHeader", linkHeader),
		zap.Int("finalOffset", finalOffset),
		zap.String("explanation", "Pagination will stop when currentOffset >= finalOffset"),
	)

	return finalOffset, nil
}
