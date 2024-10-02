package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPagination(t *testing.T) {
	t.Run("GetNextToken", func(t *testing.T) {
		testCases := []struct {
			message         string
			offset          int
			limit           int
			total           int
			pagingRequestId string
			expected        string
		}{
			{
				message:         "next page",
				offset:          80,
				limit:           10,
				total:           95,
				pagingRequestId: "",
				expected:        "{\"pagingRequestId\":\"\",\"offset\":90}",
			},
			{
				message:         "no more results",
				offset:          0,
				limit:           100,
				total:           100,
				pagingRequestId: "",
				expected:        "",
			},
			{
				message:         "pagingRequestId",
				offset:          0,
				limit:           100,
				total:           200,
				pagingRequestId: "example",
				expected:        "{\"pagingRequestId\":\"example\",\"offset\":100}",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.message, func(t *testing.T) {
				actual := GetNextToken(
					testCase.offset,
					testCase.limit,
					testCase.total,
					testCase.pagingRequestId,
				)
				require.Equal(t, testCase.expected, actual)
			})
		}
	})
}
