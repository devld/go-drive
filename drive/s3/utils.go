package s3

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"

	"go-drive/common/req"
)

// withContentMD5 is a helper function to add content MD5 to the S3 request
// see https://github.com/aws/aws-sdk-go-v2/discussions/2960
func withContentMD5(o *s3.Options) {
	o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
		stack.Initialize.Remove("AWSChecksum:SetupInputContext")
		stack.Build.Remove("AWSChecksum:RequestMetricsTracking")
		stack.Finalize.Remove("AWSChecksum:ComputeInputPayloadChecksum")
		stack.Finalize.Remove("addInputChecksumTrailer")
		return smithyhttp.AddContentChecksumMiddleware(stack)
	})
}

func newServerHTTPClient(base aws.HTTPClient, headers http.Header) *http.Client {
	if base == nil {
		base = awshttp.NewBuildableClient()
	}
	next := http.RoundTripper(httpClientRoundTripper{client: base})
	if len(headers) > 0 {
		next = requestHeaderRoundTripper{headers: headers, base: next}
	}
	return req.WithDefaultRoundTripper(&http.Client{
		Transport: next,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	})
}

type httpClientRoundTripper struct {
	client aws.HTTPClient
}

func (t httpClientRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return t.client.Do(r)
}

// requestHeaderRoundTripper applies configured headers after the default
// User-Agent, and after S3 has signed the request. User-Agent and other
// unsigned identifying headers therefore keep the configured value.
type requestHeaderRoundTripper struct {
	headers http.Header
	base    http.RoundTripper
}

func (t requestHeaderRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	for name, values := range t.headers {
		r2.Header.Del(name)
		for _, value := range values {
			r2.Header.Add(name, value)
		}
	}
	return t.base.RoundTrip(r2)
}
