package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
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
	RetryAttemptsMaximum          = 1000
	ReportLookBackDefault         = 10 * time.Hour * 24 * 365 // 10 years
	RetryAfterDefault             = 10 * time.Second          // 10 seconds
)

type Client struct {
	baseUrl        *url.URL
	bearerToken    string
	Cache          StatusesStore
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

	return &Client{
		Cache:          make(map[string]map[string]string),
		baseUrl:        parsedUrl,
		bearerToken:    token,
		organizationId: organizationId,
		wrapper:        uhttp.NewBaseHttpClient(httpClient),
	}, nil
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

	totalString := response.Header.Get(HeaderNameTotalCount)
	total, err := strconv.Atoi(totalString)
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
	query := map[string]interface{}{
		"max":    limit,
		"offset": offset,
		// blank pagingRequestId is ignored.
		"pagingRequestId": pagingRequestId,
	}
	var target []Course
	response, ratelimitData, err := c.get(ctx, ApiPathCoursesList, query, &target)
	if err != nil {
		return nil, "", 0, ratelimitData, err
	}
	defer response.Body.Close()

	totalString := response.Header.Get(HeaderNameTotalCount)
	total, err := strconv.Atoi(totalString)
	if err != nil {
		return nil, "", 0, ratelimitData, err
	}

	pagingRequestId = response.Header.Get(HeaderNamePagingRequestId)
	return target, pagingRequestId, total, ratelimitData, nil
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
		Start: now.Add(-ReportLookBackDefault),
		End:   now,
		// TODO MARCOS 1 pick default configurations.
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

	for i := 0; i < RetryAttemptsMaximum; i++ {
		// While the report is still processing, we get this ReportStatus
		// object. Once we actually get data, it'll return an array of rows.
		response, ratelimitData0, err := c.get(
			ctx,
			// Punt setting `organizationId`, it is added in `doRequest()`.
			fmt.Sprintf(ApiPathReport, "%s", c.ReportStatus.Id),
			nil,
			&target,
		)
		ratelimitData = ratelimitData0
		if err != nil {
			if response == nil {
				return nil, fmt.Errorf("got no response")
			}
			// If we got an error unmarshalling, it might be because the report
			// is still being generated. If that's the case, try unmarshalling
			// with the expected shape.
			var bodyBytes []byte
			_, err := response.Body.Read(bodyBytes)
			if err != nil {
				return nil, err
			}

			err = json.Unmarshal(bodyBytes, &c.ReportStatus)
			if err != nil {
				return nil, err
			}

			time.Sleep(RetryAfterDefault)
			continue
		}

		// We got the report object.
		defer response.Body.Close()
		break
	}

	c.ReportStatus.Status = "done"
	err := c.Cache.Load(target)
	if err != nil {
		return nil, err
	}
	return ratelimitData, nil
}
