package s3

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
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
