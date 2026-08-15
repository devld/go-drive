package s3

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type requestHeadersMiddleware struct {
	headers http.Header
}

const requestHeadersMiddlewareID = "go-drive:RequestHeaders"

func (*requestHeadersMiddleware) ID() string {
	return requestHeadersMiddlewareID
}

func (m *requestHeadersMiddleware) HandleBuild(ctx context.Context, in middleware.BuildInput,
	next middleware.BuildHandler) (middleware.BuildOutput, middleware.Metadata, error) {
	request, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return middleware.BuildOutput{}, middleware.Metadata{}, fmt.Errorf("unknown transport type %T", in.Request)
	}
	for name, values := range m.headers {
		request.Header.Del(name)
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	return next.HandleBuild(ctx, in)
}

func withRequestHeaders(headers http.Header) func(*middleware.Stack) error {
	return func(stack *middleware.Stack) error {
		if len(headers) == 0 {
			return nil
		}
		return stack.Build.Add(&requestHeadersMiddleware{headers: headers.Clone()}, middleware.After)
	}
}

// withoutRequestHeaders explicitly removes custom headers from presigned
// requests because direct browser transfers do not send them.
func withoutRequestHeaders(o *s3.Options) {
	o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
		stack.Build.Remove(requestHeadersMiddlewareID)
		return nil
	})
}

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
