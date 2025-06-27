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

// getUrl method constructs a full URL for an API request.
// It implements the URL building logic required by all API call methods.
// The method takes a relative path and a map of query parameters, combines them with the client's base URL and organization ID, and returns a complete `net/url.URL` object.
// Which centralizes URL construction, ensuring consistency across all outgoing requests.
// This implementation handles type conversion for common query parameter types like string, int, and bool.
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

// WithBearerToken function creates a `uhttp.RequestOption` to add an Authorization header.
// It implements a reusable request option for authenticating with the Percipio API.
// The function takes a bearer token string and returns a `uhttp.RequestOption` that sets the `Authorization` header.
// Which provides a clean and reusable way to add authentication to every API request.
// This implementation is a simple wrapper around `uhttp.WithHeader` for a common authentication pattern.
func WithBearerToken(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", fmt.Sprintf("Bearer %s", token))
}

// get method performs a GET request to a specified API path.
// It implements a generic helper for making GET requests, used by functions like `GetUsers` and `GetCourses`.
// The method wraps the more generic `doRequest` function, setting the HTTP method to GET and passing through the path, parameters, and target struct.
// Which simplifies the process of making GET requests within the client.
// This implementation acts as a convenient shorthand for `doRequest` with `http.MethodGet`.
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

// post method performs a POST request to a specified API path.
// It implements a generic helper for making POST requests, used by `GenerateLearningActivityReport`.
// The method wraps the more generic `doRequest` function, setting the HTTP method to POST and passing through the path, body, and target struct.
// Which simplifies the process of making POST requests within the client.
// This implementation acts as a convenient shorthand for `doRequest` with `http.MethodPost`.
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

// doRequest method is the central function for executing all HTTP requests.
// It implements the core request logic for the Percipio client, used by `get` and `post` helpers.
// The method constructs the full URL, sets up request options (including authentication and body payload), creates the request, and executes it using the `uhttp.BaseHttpClient`.
// Which ensures that all outgoing API calls are handled consistently, with proper headers, authentication, and error handling.
// This implementation leverages the `baton-sdk/pkg/uhttp` package to handle low-level request execution, response parsing, and rate limit data extraction.
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
