package client

import (
	"context"
	"net/url"
	"strconv"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const (
	BaseApiUrl           = "https://api.percipio.com"
	UsersListApiPath     = "/user-management/v1/organizations/%s/users"
	CoursesListApiPath   = "/content-discovery/v2/organizations/%s/catalog-content"
	PageSizeDefault      = 1000
	HeaderNameTotalCount = "x-total-count"
)

type Client struct {
	baseUrl        *url.URL
	bearerToken    string
	organizationId string
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
		"limit":  limit,
		"offset": offset,
	}
	var target []User
	response, ratelimitData, err := c.get(ctx, UsersListApiPath, query, &target)
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

func (c *Client) GetCourses(
	ctx context.Context,
	offset int,
	limit int,
) (
	[]Course,
	int,
	*v2.RateLimitDescription,
	error,
) {
	query := map[string]interface{}{
		"limit":  limit,
		"offset": offset,
	}
	var target []Course
	response, ratelimitData, err := c.get(ctx, CoursesListApiPath, query, &target)
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
