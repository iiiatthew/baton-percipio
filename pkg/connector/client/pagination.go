package client

import (
	"encoding/json"

	"github.com/conductorone/baton-sdk/pkg/pagination"
	"go.uber.org/zap"
)

const (
	// Emergency safety limit to prevent infinite pagination
	MaxPagesPerSync = 100 // This should handle up to 100,000 courses (100 * 1000)
)

type Pagination struct {
	PagingRequestId string `json:"pagingRequestId"`
	Offset          int    `json:"offset"`
}

// SimplePagination for endpoints that only use offset/max (like users)
type SimplePagination struct {
	Offset int `json:"offset"`
}

// ParsePaginationToken - takes a pagination token and returns offset, limit,
// and `pagingRequestId` in that order. Used for COURSES (content-discovery) endpoint.
func ParsePaginationToken(pToken *pagination.Token) (
	int,
	int,
	string,
	error,
) {
	logger := zap.L() // Get global logger

	var (
		limit           = PageSizeDefault
		offset          = 0
		pagingRequestId = ""
	)

	if pToken == nil {
		logger.Debug("ParsePaginationToken: nil token, using defaults",
			zap.Int("defaultLimit", limit),
			zap.Int("defaultOffset", offset),
		)
		return offset, limit, pagingRequestId, nil
	}

	logger.Debug("ParsePaginationToken called",
		zap.String("token", pToken.Token),
		zap.Int("size", pToken.Size),
	)

	if pToken.Size > 0 {
		limit = pToken.Size
	}

	if pToken.Token != "" {
		var parsed Pagination
		err := json.Unmarshal([]byte(pToken.Token), &parsed)
		if err != nil {
			logger.Error("ParsePaginationToken: failed to unmarshal token",
				zap.String("token", pToken.Token),
				zap.Error(err),
			)
			return 0, 0, "", err
		}
		offset = parsed.Offset
		pagingRequestId = parsed.PagingRequestId
	}

	logger.Debug("ParsePaginationToken result",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.String("pagingRequestId", pagingRequestId),
	)

	return offset, limit, pagingRequestId, nil
}

// ParseSimplePaginationToken - takes a pagination token and returns offset, limit.
// Used for USERS (user-management) endpoint that doesn't use pagingRequestId.
func ParseSimplePaginationToken(pToken *pagination.Token) (
	int,
	int,
	error,
) {
	logger := zap.L()

	var (
		limit  = PageSizeDefault
		offset = 0
	)

	if pToken == nil {
		logger.Debug("ParseSimplePaginationToken: nil token, using defaults",
			zap.Int("defaultLimit", limit),
			zap.Int("defaultOffset", offset),
		)
		return offset, limit, nil
	}

	logger.Debug("ParseSimplePaginationToken called",
		zap.String("token", pToken.Token),
		zap.Int("size", pToken.Size),
	)

	if pToken.Size > 0 {
		limit = pToken.Size
	}

	if pToken.Token != "" {
		var parsed SimplePagination
		err := json.Unmarshal([]byte(pToken.Token), &parsed)
		if err != nil {
			logger.Error("ParseSimplePaginationToken: failed to unmarshal token",
				zap.String("token", pToken.Token),
				zap.Error(err),
			)
			return 0, 0, err
		}
		offset = parsed.Offset
	}

	logger.Debug("ParseSimplePaginationToken result",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
	)

	return offset, limit, nil
}

// GetNextToken given a limit, offset, and `pagingRequestId` that were used to
// fetch _this_ page of data, and total number of resources, return the next
// pagination token as a string. Used for COURSES (content-discovery) endpoint.
func GetNextToken(
	offset int,
	limit int,
	total int,
	pagingRequestId string,
) string {
	logger := zap.L()
	nextOffset := offset + limit
	currentPage := (offset / limit) + 1

	logger.Debug("GetNextToken called",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.Int("total", total),
		zap.Int("nextOffset", nextOffset),
		zap.Int("currentPage", currentPage),
		zap.String("pagingRequestId", pagingRequestId),
	)

	// Emergency safety check: prevent infinite pagination
	if currentPage >= MaxPagesPerSync {
		logger.Warn("GetNextToken: pagination safety limit reached",
			zap.Int("currentPage", currentPage),
			zap.Int("maxPagesPerSync", MaxPagesPerSync),
			zap.Int("totalCoursesProcessed", offset),
		)
		return "" // Force pagination to stop
	}

	if nextOffset >= total {
		logger.Debug("GetNextToken: pagination complete",
			zap.Int("nextOffset", nextOffset),
			zap.Int("total", total),
			zap.Int("totalPages", currentPage),
		)
		return ""
	}

	bytes, err := json.Marshal(
		Pagination{
			Offset:          nextOffset,
			PagingRequestId: pagingRequestId,
		},
	)
	if err != nil {
		logger.Error("GetNextToken: failed to marshal pagination token",
			zap.Int("nextOffset", nextOffset),
			zap.String("pagingRequestId", pagingRequestId),
			zap.Error(err),
		)
		return ""
	}

	nextToken := string(bytes)
	logger.Debug("GetNextToken: token generated",
		zap.String("nextToken", nextToken),
		zap.Int("nextPage", currentPage+1),
		zap.Float64("progressPercent", float64(nextOffset)/float64(total)*100),
	)

	return nextToken
}

// GetSimpleNextToken given a limit, offset, and total number of resources,
// return the next pagination token as a string. Used for USERS (user-management) endpoint.
func GetSimpleNextToken(
	offset int,
	limit int,
	total int,
) string {
	logger := zap.L()
	nextOffset := offset + limit
	currentPage := (offset / limit) + 1

	logger.Debug("GetSimpleNextToken called",
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.Int("total", total),
		zap.Int("nextOffset", nextOffset),
		zap.Int("currentPage", currentPage),
	)

	// Emergency safety check: prevent infinite pagination
	if currentPage >= MaxPagesPerSync {
		logger.Warn("GetSimpleNextToken: pagination safety limit reached",
			zap.Int("currentPage", currentPage),
			zap.Int("maxPagesPerSync", MaxPagesPerSync),
			zap.Int("totalUsersProcessed", offset),
		)
		return "" // Force pagination to stop
	}

	if nextOffset >= total {
		logger.Debug("GetSimpleNextToken: pagination complete",
			zap.Int("nextOffset", nextOffset),
			zap.Int("total", total),
			zap.Int("totalPages", currentPage),
		)
		return ""
	}

	bytes, err := json.Marshal(
		SimplePagination{
			Offset: nextOffset,
		},
	)
	if err != nil {
		logger.Error("GetSimpleNextToken: failed to marshal pagination token",
			zap.Int("nextOffset", nextOffset),
			zap.Error(err),
		)
		return ""
	}

	nextToken := string(bytes)
	logger.Debug("GetSimpleNextToken: token generated",
		zap.String("nextToken", nextToken),
		zap.Int("nextPage", currentPage+1),
		zap.Float64("progressPercent", float64(nextOffset)/float64(total)*100),
	)

	return nextToken
}
