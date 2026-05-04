package agent

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

func NewHTTPClient(
	addr string,
	timeout time.Duration,
	retryCount int,
	waitTime time.Duration,
	maxWaitTime time.Duration,
) *resty.Client {
	return resty.New().
		SetBaseURL("http://" + addr).
		SetTimeout(timeout).
		SetRetryCount(retryCount).
		SetRetryWaitTime(waitTime).
		SetRetryMaxWaitTime(maxWaitTime).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			if err != nil {
				return isRetryableNetErr(err)
			}

			if resp == nil {
				return false
			}

			return resp.StatusCode() >= http.StatusInternalServerError
		})
}
