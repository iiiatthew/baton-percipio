package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/conductorone/baton-percipio/pkg/connector/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

func AssertNoRatelimitAnnotations(
	t *testing.T,
	actualAnnotations annotations.Annotations,
) {
	if actualAnnotations != nil && len(actualAnnotations) == 0 {
		return
	}

	for _, annotation := range actualAnnotations {
		var ratelimitDescription v2.RateLimitDescription
		err := annotation.UnmarshalTo(&ratelimitDescription)
		if err != nil {
			continue
		}
		if slices.Contains(
			[]v2.RateLimitDescription_Status{
				v2.RateLimitDescription_STATUS_ERROR,
				v2.RateLimitDescription_STATUS_OVERLIMIT,
			},
			ratelimitDescription.Status,
		) {
			t.Fatal("request was ratelimited, expected not to be ratelimited")
		}
	}
}

func FixturesServer() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(
			func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set(uhttp.ContentType, "application/json")
				writer.Header().Set(client.HeaderNameTotalCount, "2")
				writer.Header().Set(client.HeaderNamePagingRequestId, "test-paging-id")

				var filename string
				routeUrl := request.URL.String()
				switch {
				case strings.Contains(routeUrl, "search-content"):
					filename = "../../test/fixtures/search_content.json"
				case strings.Contains(routeUrl, "report-requests/learning-activity"):
					filename = "../../test/fixtures/reportStatus0.json"
				case strings.Contains(routeUrl, "report-requests/"):
					filename = "../../test/fixtures/report.json"
				case strings.Contains(routeUrl, "catalog"):
					// Add mock link header for content pagination testing
					linkHeader := "</v2/organizations/test-org/catalog-content?offset=0&max=1000&pagingRequestId=test-paging-id>; " +
						"page=\"1\"; per_page=\"1000\"; rel=\"first\", " +
						"</v2/organizations/test-org/catalog-content?offset=2000&max=1000&pagingRequestId=test-paging-id>; " +
						"page=\"3\"; per_page=\"1000\"; rel=\"last\""
					writer.Header().Set("link", linkHeader)
					filename = "../../test/fixtures/courses0.json"
				case strings.Contains(routeUrl, "users"):
					filename = "../../test/fixtures/users0.json"
				default:
					// This should never happen in tests.
					panic(fmt.Errorf("bad url: %s", routeUrl))
				}
				writer.WriteHeader(http.StatusOK)
				data, _ := os.ReadFile(filename)
				_, err := writer.Write(data)
				if err != nil {
					return
				}
			},
		),
	)
}
