package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/conductorone/baton-percipio/pkg/config"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	ApiPathCoursesList            = "/content-discovery/v2/organizations/%s/catalog-content"
	ApiPathLearningActivityReport = "/reporting/v1/organizations/%s/report-requests/learning-activity"
	ApiPathReport                 = "/reporting/v1/organizations/%s/report-requests/%s"
	ApiPathUsersList              = "/user-management/v1/organizations/%s/users"
	BaseApiUrl                    = "https://api.percipio.com"
	HeaderNamePagingRequestId     = "x-paging-request-id"
	HeaderNameTotalCount          = "x-total-count"
	PageSizeDefault               = 1000
	ReportLookBackDefault         = 10 * time.Hour * 24 * 365 // 10 years
)

type Client struct {
	baseUrl        *url.URL
	bearerToken    string
	StatusesStore  StatusesStore
	organizationId string
	ReportStatus   ReportStatus
	wrapper        *uhttp.BaseHttpClient
}

func New(
	ctx context.Context,
	baseUrl string,
	organizationId string,
	token string,
) (*Client, error) {
	httpClient, err := uhttp.NewClient(
		ctx,
		uhttp.WithLogger(
			true,
			ctxzap.Extract(ctx),
		),
	)
	if err != nil {
		return nil, err
	}

	parsedUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, err
	}

	wrapper, err := uhttp.NewBaseHttpClientWithContext(ctx, httpClient)
	if err != nil {
		return nil, err
	}

	return &Client{
		StatusesStore:  make(map[string]map[string]string),
		baseUrl:        parsedUrl,
		bearerToken:    token,
		organizationId: organizationId,
		wrapper:        wrapper,
	}, nil
}

func getTotalCount(response *http.Response) (int, error) {
	totalString := response.Header.Get(HeaderNameTotalCount)
	return strconv.Atoi(totalString)
}

// GetUsers returns
// - a page of users
// - the reported total number of users that match the filter criteria
// - any ratelimit data
// - an error.
func (c *Client) GetUsers(
	ctx context.Context,
	offset int,
	limit int,
) (
	[]User,
	int,
	*v2.RateLimitDescription,
	error,
) {
	query := map[string]interface{}{
		"max":    limit,
		"offset": offset,
	}
	var target []User
	response, ratelimitData, err := c.get(ctx, ApiPathUsersList, query, &target)
	if err != nil {
		return nil, 0, ratelimitData, err
	}
	defer response.Body.Close()

	total, err := getTotalCount(response)
	if err != nil {
		return nil, 0, ratelimitData, err
	}
	return target, total, ratelimitData, nil
}

// GetCourses fetches courses using offset-based pagination.
// Returns courses, pagingRequestId, finalOffset, ratelimit, error.
func (c *Client) GetCourses(
	ctx context.Context,
	offset int,
	limit int,
	pagingRequestId string,
) (
	[]Course,
	string,
	int,
	*v2.RateLimitDescription,
	error,
) {
	// Always use offset/max parameters for all calls
	query := map[string]interface{}{
		"max":    limit,
		"offset": offset,
	}

	// Add pagingRequestId for subsequent calls
	if pagingRequestId != "" {
		query["pagingRequestId"] = pagingRequestId
	}

	var target []Course
	response, ratelimitData, err := c.get(ctx, ApiPathCoursesList, query, &target)
	if err != nil {
		return nil, "", 0, ratelimitData, err
	}
	defer response.Body.Close()

	newPagingRequestId := response.Header.Get(HeaderNamePagingRequestId)

	var finalOffset int
	if pagingRequestId == "" {
		// First call: parse link header to get finalOffset
		linkHeader := response.Header.Get("link")
		if linkHeader != "" {
			finalOffset, err = ParseLinkHeader(linkHeader)
			if err != nil {
				return nil, "", 0, ratelimitData, fmt.Errorf("failed to parse link header: %w", err)
			}
		}
	} else {
		// Subsequent calls: finalOffset not needed from response
		finalOffset = 0
	}

	return target, newPagingRequestId, finalOffset, ratelimitData, nil
}

// GenerateLearningActivityReport makes a post request to the API asking it to start generating a report. We'll need to then poll a _different_ endpoint to get the actual report data.
func (c *Client) GenerateLearningActivityReport(
	ctx context.Context,
) (
	*v2.RateLimitDescription,
	error,
) {
	now := time.Now()
	body := ReportConfigurations{
		End:         now,
		Start:       now.Add(-ReportLookBackDefault),
		ContentType: "Course,Assessment",
	}

	var target ReportStatus
	response, ratelimitData, err := c.post(
		ctx,
		ApiPathLearningActivityReport,
		body,
		&target,
	)
	if err != nil {
		return ratelimitData, err
	}
	defer response.Body.Close()

	// Should include ID and "PENDING".
	c.ReportStatus = target

	return ratelimitData, nil
}

func (c *Client) pollLearningActivityReport(ctx context.Context, reportUrl string) ([]byte, *v2.RateLimitDescription, error) {
	var ratelimitData *v2.RateLimitDescription

	l := ctxzap.Extract(ctx)
	for i := 0; i < config.RetryAttemptsMaximum; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reportUrl, nil)
		if err != nil {
			return nil, nil, err
		}

		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, nil, err
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		// We can ignore this error and proceed with parsing the body.
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			l.Error("error reading response body", zap.Error(err))
			_ = resp.Body.Close()
			return nil, ratelimitData, err
		}
		_ = resp.Body.Close()

		// Trim whitespace to check the first character.
		trimmedBody := bytes.TrimSpace(bodyBytes)
		if len(trimmedBody) == 0 {
			l.Warn("empty response body from percipio api, retrying...")
			time.Sleep(time.Second * time.Duration(config.RetryAfterSeconds))
			continue
		}

		// If the response is a JSON array, it's the report.
		if trimmedBody[0] == '[' {
			return trimmedBody, ratelimitData, nil
		}

		// If the response is a JSON object, it's a status update.
		if trimmedBody[0] == '{' {
			var reportStatus ReportStatus
			err = json.Unmarshal(trimmedBody, &reportStatus)
			if err != nil {
				l.Error("error unmarshalling report status", zap.Error(err), zap.String("body", string(trimmedBody)))
				// If we can't unmarshal the status, it might be the report data, but it started with a '{'.
				// This is an unexpected state. It is safer to return an error than to return potentially invalid data.
				return nil, ratelimitData, fmt.Errorf("failed to unmarshal report status object: %w", err)
			}

			l.Debug("report status",
				zap.String("status", reportStatus.Status),
				zap.Int("attempt", i),
				zap.Int("retry_after_seconds", config.RetryAfterSeconds),
				zap.Int("retry_attempts_maximum", config.RetryAttemptsMaximum))

			if reportStatus.Status == "PENDING" || reportStatus.Status == "IN_PROGRESS" {
				time.Sleep(time.Second * time.Duration(config.RetryAfterSeconds))
				continue
			}

			// Any other status is treated as a failure.
			return nil, ratelimitData, fmt.Errorf("report generation failed with status: %s", reportStatus.Status)
		}

		// If it's neither, the response is unexpected.
		return nil, ratelimitData, fmt.Errorf("unexpected report response format")
	}

	return nil, ratelimitData, fmt.Errorf("report polling timed out")
}

func (c *Client) GetLearningActivityReport(
	ctx context.Context,
) (
	*v2.RateLimitDescription,
	error,
) {
	var (
		ratelimitData *v2.RateLimitDescription
		target        Report
	)
	reportUrl := fmt.Sprintf("%s%s", c.baseUrl.String(), fmt.Sprintf(ApiPathReport, c.organizationId, c.ReportStatus.Id))
	bodyBytes, ratelimitData, err := c.pollLearningActivityReport(ctx, reportUrl)
	if err != nil {
		return ratelimitData, err
	}

	l := ctxzap.Extract(ctx)
	err = json.Unmarshal(bodyBytes, &target)
	if err != nil {
		l.Error("error unmarshalling learning activity report", zap.Error(err))
		return ratelimitData, err
	}

	c.ReportStatus.Status = "COMPLETED"

	l.Debug("loading report")
	err = c.StatusesStore.Load(&target)
	if err != nil {
		return ratelimitData, err
	}
	return ratelimitData, nil
}
