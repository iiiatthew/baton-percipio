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
	ApiPathSearchContent          = "/content-discovery/v1/organizations/%s/search-content"
	ApiPathLearningActivityReport = "/reporting/v1/organizations/%s/report-requests/learning-activity"
	ApiPathReport                 = "/reporting/v1/organizations/%s/report-requests/%s"
	ApiPathUsersList              = "/user-management/v1/organizations/%s/users"
	BaseApiUrl                    = "https://api.percipio.com"
	HeaderNamePagingRequestId     = "x-paging-request-id"
	HeaderNameTotalCount          = "x-total-count"
	PageSizeDefault               = 1000
	ReportLookBackDefault         = 10 * time.Hour * 24 * 365 // 10 years
)

// Client struct manages all communication with the Percipio API.
// It is used by the connector to abstract away the details of HTTP requests and response handling.
// It holds fields such as baseUrl, bearerToken, and organizationId for authenticating and targeting API calls.
// This structure organizes API client configuration and stateful data like ReportStatus for multi-step report generation.
// Instances are typically created by the New function and populated with configuration from the connector.
type Client struct {
	baseUrl        *url.URL
	bearerToken    string
	StatusesStore  StatusesStore
	organizationId string
	ReportStatus   ReportStatus
	wrapper        *uhttp.BaseHttpClient
}

// New function creates and initializes a new Percipio API Client.
// It implements the instantiation of the API client required by the connector to interact with the Percipio API.
// The client is created by configuring a `uhttp.Client` from the baton-sdk, parsing the provided base URL, and populating the Client struct with authentication details.
// Which provides a centralized and consistent method for creating a ready-to-use API client.
// This implementation aligns with SDK patterns by using `uhttp.NewClient` for robust, logged HTTP communication.
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

// getTotalCount function extracts the total result count from an HTTP response.
// It implements the parsing of the `x-total-count` header, which is expected from Percipio's paginated API endpoints.
// The function reads the `HeaderNameTotalCount` constant value from the response header and converts it to an integer.
// Which provides the total number of available records, a crucial piece of information for managing pagination logic.
// This implementation is a straightforward helper to centralize a common API-specific parsing task.
func getTotalCount(response *http.Response) (int, error) {
	totalString := response.Header.Get(HeaderNameTotalCount)
	return strconv.Atoi(totalString)
}

// GetUsers method fetches a single page of user resources from the Percipio API.
// It implements the user data retrieval operation required by the user resource syncer.
// The method builds a query with offset and limit parameters and calls the internal `get` helper
// to execute the request against the `ApiPathUsersList` endpoint.
// Which enables the connector to paginate through the entire set of users in the Percipio tenant.
// This implementation encapsulates the logic for interacting with the user management endpoint.
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

// GetCourses method fetches a single page of course resources using Percipio's specialized catalog pagination.
// It implements the content data retrieval operation required by the course resource syncer for a full sync.
// The method manages a stateful pagination flow by sending an `offset` and `limit`, and then using a `pagingRequestId` returned in the first response for all subsequent requests.
// Which is the core operation for retrieving all available course and assessment resources from the Percipio tenant.
// This implementation is tailored to the non-standard pagination of the `/catalog-content` endpoint.
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
		return nil, "", 0, ratelimitData, err
	}
	defer response.Body.Close()

	newPagingRequestId := response.Header.Get(HeaderNamePagingRequestId)

	var finalOffset int
	if pagingRequestId == "" {
		linkHeader := response.Header.Get("link")
		if linkHeader != "" {
			finalOffset, err = ParseLinkHeader(ctx, linkHeader)
			if err != nil {
				return nil, "", 0, ratelimitData, fmt.Errorf("failed to parse link header: %w", err)
			}
		}
	} else {
		finalOffset = 0
	}

	return target, newPagingRequestId, finalOffset, ratelimitData, nil
}

// SearchContentByID function searches for a single course or assessment by its unique ID.
// It implements a more targeted content retrieval method required for the limited-courses sync feature.
// The function constructs a GET request to the `/search-content` endpoint using the content ID as a query and filters for COURSE and ASSESSMENT types.
// Which provides an efficient way to fetch specific content items without paginating through the entire catalog.
// This implementation makes a separate API call for each ID because the Percipio search API may return unexpected or overly broad results if more than one ID is queried at a time.
func (c *Client) SearchContentByID(
	ctx context.Context,
	courseID string,
) (
	[]Course,
	*v2.RateLimitDescription,
	error,
) {
	query := map[string]interface{}{
		"q":          courseID,
		"typeFilter": "COURSE,ASSESSMENT",
	}

	var target []Course
	response, ratelimitData, err := c.get(ctx, ApiPathSearchContent, query, &target)
	if err != nil {
		return nil, ratelimitData, err
	}
	defer response.Body.Close()

	return target, ratelimitData, nil
}

// GenerateLearningActivityReport method initiates the creation of a learning activity report.
// It implements the first step of the asynchronous report generation process required by the connector to fetch grants.
// The method sends a POST request to the `ApiPathLearningActivityReport` endpoint with a lookback period, which triggers a background job on the Percipio service.
// Which is the only way the connector can access data about user course assignments, completions, and progress.
// This implementation stores the returned report ID in the `c.ReportStatus` field, which is essential for the subsequent polling step.
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

	c.ReportStatus = target

	return ratelimitData, nil
}

// pollLearningActivityReport method polls a report URL until the report is successfully generated.
// The function makes repeated GET requests to the report URL until the status is no longer "IN_PROGRESS".
// Which is necessary because the initial report generation request only returns a job ID, not the final data.
// This implementation includes a custom retry loop and handles the API's unusual behavior
// of returning different data structures for the same endpoint.
// We use the native Go net/http package instead of uhttp for the report polling function as uhttp
// seems to ignore Cache-Control: no-cache headers and kept returning IN_PROGRESS for the report polling
// even when the report was completed and available during testing.
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
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			l.Error("error reading response body", zap.Error(err))
			_ = resp.Body.Close()
			return nil, ratelimitData, err
		}
		_ = resp.Body.Close()

		trimmedBody := bytes.TrimSpace(bodyBytes)
		if len(trimmedBody) == 0 {
			l.Warn("empty response body from percipio api, retrying...")
			time.Sleep(time.Second * time.Duration(config.RetryAfterSeconds))
			continue
		}

		if trimmedBody[0] == '[' {
			return trimmedBody, ratelimitData, nil
		}

		if trimmedBody[0] == '{' {
			var reportStatus ReportStatus
			err = json.Unmarshal(trimmedBody, &reportStatus)
			if err != nil {
				l.Error("error unmarshalling report status", zap.Error(err), zap.String("body", string(trimmedBody)))
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

			return nil, ratelimitData, fmt.Errorf("report generation failed with status: %s", reportStatus.Status)
		}

		return nil, ratelimitData, fmt.Errorf("unexpected report response format")
	}

	return nil, ratelimitData, fmt.Errorf("report polling timed out")
}

// GetLearningActivityReport method retrieves the completed learning activity report.
// It implements the final step of the grant data retrieval process, required by the course grant builder.
// The method first calls `pollLearningActivityReport` to wait for and receive the raw report data, then unmarshals it into a `Report` struct.
// Which makes the complete set of user-course relationships available to the connector.
// This implementation finishes the asynchronous workflow by loading the report data into the `StatusesStore` for efficient grant lookups.
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
