package client

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/conductorone/baton-sdk/pkg/pagination"
	"go.uber.org/zap"
)

type UserPagination struct {
	Offset int `json:"offset"`
}

type ContentPagination struct {
	Page            int    `json:"page"`
	PagingRequestId string `json:"pagingRequestId"`
	LastPage        int    `json:"lastPage"`
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
		page            = 1
		pagingRequestId = ""
		lastPage        = 0
	)

	if pToken != nil && pToken.Token != "" {
		var parsed ContentPagination
		err := json.Unmarshal([]byte(pToken.Token), &parsed)
		if err != nil {
			return 0, "", 0, err
		}
		page = parsed.Page
		pagingRequestId = parsed.PagingRequestId
		lastPage = parsed.LastPage
	}

	return page, pagingRequestId, lastPage, nil
}

// GetContentNextToken generates next token for content discovery API.
func GetContentNextToken(currentPage, lastPage int, pagingRequestId string) string {
	logger := zap.L()
	nextPage := currentPage + 1

	if nextPage > lastPage {
		logger.Debug("GetContentNextToken: pagination complete",
			zap.Int("currentPage", currentPage),
			zap.Int("lastPage", lastPage),
		)
		return ""
	}

	bytes, err := json.Marshal(ContentPagination{
		Page:            nextPage,
		PagingRequestId: pagingRequestId,
		LastPage:        lastPage,
	})
	if err != nil {
		logger.Error("GetContentNextToken: failed to marshal pagination token",
			zap.Int("currentPage", currentPage),
			zap.Error(err),
		)
		return ""
	}

	return string(bytes)
}

// ParseLinkHeader extracts lastPage from link header rel="last" section.
func ParseLinkHeader(linkHeader string) (int, error) {
	logger := zap.L()

	logger.Debug("ParseLinkHeader called", zap.String("linkHeader", linkHeader))

	pageRegex := regexp.MustCompile(`page="(\d+)"`)
	relRegex := regexp.MustCompile(`rel="([^"]+)"`)

	matches := pageRegex.FindAllStringSubmatch(linkHeader, -1)
	relMatches := relRegex.FindAllStringSubmatch(linkHeader, -1)

	for i, match := range matches {
		if len(match) < 2 || i >= len(relMatches) || len(relMatches[i]) < 2 {
			continue
		}

		pageNum, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}

		rel := relMatches[i][1]
		if strings.Contains(rel, "last") {
			logger.Debug("ParseLinkHeader: found lastPage",
				zap.Int("lastPage", pageNum),
				zap.String("rel", rel),
			)
			return pageNum, nil
		}
	}

	logger.Error("ParseLinkHeader: no rel=last found in link header")
	return 0, fmt.Errorf("no rel=last found in link header")
}
