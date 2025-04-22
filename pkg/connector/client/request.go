package client

import (
	"context"
	"fmt"
	"net/http"
	liburl "net/url"
	"strconv"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

func (c *Client) getUrl(
	path string,
	queryParameters map[string]any,
) *liburl.URL {
	params := liburl.Values{}
	for key, valueAny := range queryParameters {
		switch value := valueAny.(type) {
		case string:
			params.Add(key, value)
		case int:
			params.Add(key, strconv.Itoa(value))
		case bool:
			params.Add(key, strconv.FormatBool(value))
		default:
			continue
		}
	}

	output := c.baseUrl.JoinPath(fmt.Sprintf(path, c.organizationId))
	output.RawQuery = params.Encode()
	return output
}

// WithBearerToken - TODO(marcos): move this function to `baton-sdk`.
func WithBearerToken(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

func (c *Client) get(
	ctx context.Context,
	path string,
	queryParameters map[string]any,
	target any,
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequest(
		ctx,
		http.MethodGet,
		path,
		queryParameters,
		nil,
		&target,
	)
}

func (c *Client) post(
	ctx context.Context,
	path string,
	body interface{},
	target interface{},
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	return c.doRequest(
		ctx,
		http.MethodPost,
		path,
		nil,
		body,
		&target,
	)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	queryParameters map[string]any,
	payload any,
	target any,
) (
	*http.Response,
	*v2.RateLimitDescription,
	error,
) {
	options := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		WithBearerToken(c.bearerToken),
	}
	if payload != nil {
		options = append(options, uhttp.WithJSONBody(payload))
	}

	url := c.getUrl(path, queryParameters)

	request, err := c.wrapper.NewRequest(ctx, method, url, options...)
	if err != nil {
		return nil, nil, err
	}

	var ratelimitData v2.RateLimitDescription
	response, err := c.wrapper.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
		uhttp.WithJSONResponse(target),
	)
	if err != nil {
		return response, &ratelimitData, fmt.Errorf("error making %s request to %s: %w", method, url, err)
	}

	return response, &ratelimitData, nil
}
