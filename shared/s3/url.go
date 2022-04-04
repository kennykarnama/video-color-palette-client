package s3

import (
	"net/url"
	"strings"
)

// KeyEscape escapes a key according to S3 path escaping rule to be used in the S3 URL.
// Useful if you need to build an S3 URL given certain region, bucket, and key.
// Similar with url.PathEscape for building a URL but will not escape /
// The usual HTTP rule encodes space (" ") as %20 and plus ("+") as + (unchanged).
// However, the S3 escaping rule encodes space (" ") as + and plus ("+") as %2B, similar like querystring encoding.
func KeyEscape(key string) string {
	pathComponents := strings.Split(key, "/")
	results := make([]string, 0, len(pathComponents))
	for _, component := range pathComponents {
		results = append(results, url.QueryEscape(component))
	}
	return strings.Join(results, "/")
}

