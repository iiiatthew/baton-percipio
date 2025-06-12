package client

import (
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
	ReportLookBackDefault         = 10 * 24 * time.Hour // 10 days
	// ReportLookBackDefault         = 10 * time.Hour * 24 * 365 // 10 years
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
	if totalString == "" {
		return 0, fmt.Errorf("missing %s header in response", HeaderNameTotalCount)
	}

	total, err := strconv.Atoi(totalString)
	if err != nil {
		return 0, fmt.Errorf("invalid %s header value '%s': %w", HeaderNameTotalCount, totalString, err)
	}

	// Sanity check: if total is suspiciously high, something is wrong
	if total > 1000000 { // More than 1 million courses seems suspicious
		return 0, fmt.Errorf("suspiciously high total count %d from %s header", total, HeaderNameTotalCount)
	}

	return total, nil
}

// getTotalCountForContentDiscovery handles the case where content-discovery API
// may not provide x-total-count header (not documented in swagger)
func getTotalCountForContentDiscovery(response *http.Response, coursesReturned int, offset int, limit int) (int, error) {
	totalString := response.Header.Get(HeaderNameTotalCount)

	// If no header is provided, estimate based on returned data
	if totalString == "" {
		// If we got fewer courses than requested, we're at the end
		if coursesReturned < limit {
			estimatedTotal := offset + coursesReturned
			return estimatedTotal, nil
		}
		// Otherwise, we don't know the total - set a high estimate to continue pagination
		// The safety limit will prevent infinite loops
		return offset + coursesReturned + 1000, nil
	}

	total, err := strconv.Atoi(totalString)
	if err != nil {
		return 0, fmt.Errorf("invalid %s header value '%s': %w", HeaderNameTotalCount, totalString, err)
	}

	// Sanity check: if total is suspiciously high, something is wrong
	if total > 1000000 { // More than 1 million courses seems suspicious
		return 0, fmt.Errorf("suspiciously high total count %d from %s header", total, HeaderNameTotalCount)
	}

	return total, nil
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

// GetCourses Given a limit/offset and a pagination token (see below), fetch a
// page's worth of course data. Returns _five_ values:
//  1. the list of courses
//  2. the "pagination token", required to get any page but the first.
//  3. the total number of courses, so that we don't have to fetch an extra page
//     to confirm there are no more courses.
//  4. rate limit data read from headers.
//  5. any error
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
	// Add detailed logging for API call tracking
	logger := ctxzap.Extract(ctx)
	requestId := fmt.Sprintf("req_%d", time.Now().UnixNano())
	startTime := time.Now()

	logger.Info("GetCourses API call initiated",
		zap.String("requestId", requestId),
		zap.Int("offset", offset),
		zap.Int("limit", limit),
		zap.String("pagingRequestId", pagingRequestId),
		zap.Time("startTime", startTime),
	)

	query := map[string]interface{}{
		"max":    limit,
		"offset": offset,
	}
	if pagingRequestId != "" {
		query["pagingRequestId"] = pagingRequestId
	}
	var target []Course
	response, ratelimitData, err := c.get(ctx, ApiPathCoursesList, query, &target)
	if err != nil {
		logger.Error("GetCourses API call failed",
			zap.String("requestId", requestId),
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)),
		)
		return nil, "", 0, ratelimitData, err
	}
	defer response.Body.Close()

	newPagingRequestId := response.Header.Get(HeaderNamePagingRequestId)
	total, err := getTotalCountForContentDiscovery(response, len(target), offset, limit)
	if err != nil {
		logger.Error("GetCourses header parsing failed",
			zap.String("requestId", requestId),
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)),
		)
		return nil, "", 0, ratelimitData, err
	}

	// Validate pagination request ID behavior
	if offset > 0 && pagingRequestId != "" && newPagingRequestId == "" {
		logger.Error("GetCourses pagination context lost",
			zap.String("requestId", requestId),
			zap.Int("offset", offset),
			zap.String("originalPagingRequestId", pagingRequestId),
		)
		return nil, "", 0, ratelimitData, fmt.Errorf("API lost pagingRequestId after offset %d", offset)
	}

	// Detect potential infinite loop: same courses returned repeatedly
	if len(target) == 0 && offset < total {
		logger.Error("GetCourses empty response with remaining data",
			zap.String("requestId", requestId),
			zap.Int("offset", offset),
			zap.Int("total", total),
		)
		return nil, "", 0, ratelimitData, fmt.Errorf("API returned no courses at offset %d but total is %d", offset, total)
	}

	// Calculate current page for progress tracking
	currentPage := (offset / limit) + 1

	logger.Info("GetCourses API call completed",
		zap.String("requestId", requestId),
		zap.Int("coursesReturned", len(target)),
		zap.Int("total", total),
		zap.String("nextPagingRequestId", newPagingRequestId),
		zap.Int("currentPage", currentPage),
		zap.Float64("progressPercent", float64(offset+len(target))/float64(total)*100),
		zap.Duration("duration", time.Since(startTime)),
	)

	return target, newPagingRequestId, total, ratelimitData, nil
}

// GenerateLearningActivityReport makes a post request to the API asking it to
// start generating a report. We'll need to then poll a _different_ endpoint to
// get the actual report data.
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

	l := ctxzap.Extract(ctx)
	for i := 0; i < config.RetryAttemptsMaximum; i++ {
		// While the report is still processing, we get this ReportStatus
		// object. Once we actually get data, it'll return an array of rows.
		response, ratelimitData0, err := c.get(
			ctx,
			// Punt setting `organizationId`, it is added in `doRequest()`.
			fmt.Sprintf(ApiPathReport, "%s", c.ReportStatus.Id),
			nil,
			// Don't use response body because Percipio's API closes connections early and returns EOF sometimes.
			nil,
		)
		ratelimitData = ratelimitData0
		if err != nil {
			l.Error("error getting report", zap.Error(err))
			// Ignore unexpected EOF because Precipio returns this on success sometimes
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				return ratelimitData, err
			}
		}
		if response == nil {
			return ratelimitData, fmt.Errorf("no response from precipio api")
		}

		defer response.Body.Close()
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			l.Error("error reading response body", zap.Error(err))
			return ratelimitData, err
		}

		// Response can be a report status if the report isn't done processing, or the report. Try status first.
		err = json.Unmarshal(bodyBytes, &c.ReportStatus)
		if err == nil {
			l.Debug("report status",
				zap.String("status", c.ReportStatus.Status),
				zap.Int("attempt", i),
				zap.Int("retry_after_seconds", config.RetryAfterSeconds),
				zap.Int("retry_attempts_maximum", config.RetryAttemptsMaximum))
			if c.ReportStatus.Status == "FAILED" {
				return ratelimitData, fmt.Errorf("report generation failed: %v", c.ReportStatus)
			}
			time.Sleep(config.RetryAfterSeconds * time.Second)
			continue
		}
		syntaxError := new(json.SyntaxError)
		if errors.As(err, &syntaxError) {
			l.Warn("syntax error unmarshaling report status", zap.Error(err))
			time.Sleep(config.RetryAfterSeconds * time.Second)
			continue
		}
		unmarshalError := new(json.UnmarshalTypeError)
		if !errors.As(err, &unmarshalError) {
			return ratelimitData, err
		}

		l.Debug("unmarshaling to report status failed. trying to unmarshall as report", zap.Error(err))
		err = json.Unmarshal(bodyBytes, &target)
		if err != nil {
			return ratelimitData, err
		}
		// We got the report object.
		break
	}

	c.ReportStatus.Status = "done"
	l.Debug("loading report")
	err := c.StatusesStore.Load(&target)
	if err != nil {
		return ratelimitData, err
	}
	return ratelimitData, nil
}
