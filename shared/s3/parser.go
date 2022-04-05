// Taken from https://gitlab.com/ruangguru/source/-/raw/master/shared-lib/go/client/s3/url.go
package s3

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

const GlobalDefaultRegion = "us-east-1"

// taken from https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
var (
	s3VirtualHostPattern = regexp.MustCompile(`^https://([^\.]+)\.s3\.([^\.]+)\.amazonaws\.com(/[^?^#]*)?`) // https://bucket-name.s3.Region.amazonaws.com
	s3PathPattern        = regexp.MustCompile(`^https://s3\.([^\.]+)\.amazonaws\.com/([^/]+)(/[^?^#]*)?`)   // https://s3.Region.amazonaws.com/bucket-name
	s3DashRegionPattern  = regexp.MustCompile(`^https://([^\.]+)\.s3-([^\.]+)\.amazonaws\.com(/[^?^#]*)?`)  // https://bucket-name.s3-Region.amazonaws.com
	s3GlobalPattern      = regexp.MustCompile(`^https://([^\.]+)\.s3\.amazonaws\.com(/[^?^#]*)?`)           // https://bucket-name.s3.amazonaws.com

	ErrURLPatternNotFound = errors.New("pattern of S3 URL not found")
	ErrKeyUnescape        = errors.New("error unescaping key")
)

type S3URI struct {
	Region string
	Bucket string
	Key    string
}

// ParseURL parses an HTTP S3 URL into S3 URI: region, bucket, and key.
// It handles multiple S3 URL patterns, including the legacy one.
func ParseURL(s3URL string) (*S3URI, error) {
	if match := s3VirtualHostPattern.FindStringSubmatch(s3URL); len(match) >= 4 {
		key, err := url.QueryUnescape(strings.TrimPrefix(match[3], "/"))
		if err != nil {
			return nil, ErrKeyUnescape
		}
		return &S3URI{Region: match[2], Bucket: match[1], Key: key}, nil
	} else if match := s3PathPattern.FindStringSubmatch(s3URL); len(match) >= 4 {
		key, err := url.QueryUnescape(strings.TrimPrefix(match[3], "/"))
		if err != nil {
			return nil, ErrKeyUnescape
		}
		return &S3URI{Region: match[1], Bucket: match[2], Key: key}, nil
	} else if match := s3DashRegionPattern.FindStringSubmatch(s3URL); len(match) >= 4 {
		key, err := url.QueryUnescape(strings.TrimPrefix(match[3], "/"))
		if err != nil {
			return nil, ErrKeyUnescape
		}
		return &S3URI{Region: match[2], Bucket: match[1], Key: key}, nil
	} else if match := s3GlobalPattern.FindStringSubmatch(s3URL); len(match) >= 3 {
		key, err := url.QueryUnescape(strings.TrimPrefix(match[2], "/"))
		if err != nil {
			return nil, ErrKeyUnescape
		}
		// the region will be GlobalDefaultRegion
		return &S3URI{Region: "", Bucket: match[1], Key: key}, nil
	}
	return nil, ErrURLPatternNotFound
}

// ReplaceURLIfBucketMatch replaces an S3 URL with a CDN URL if the bucket match
// In the case of s3URL is not a valid S3 URL or the bucket does not match,
// the URL is returned unchanged, similar behavior like strings utility from Golang
// The scheme (http:// or https://) is required for both S3 and CDN URL
// For multiple bucket-CDN mapping, see ReplaceURLWithBucketMapping
func ReplaceURLIfBucketMatch(s3URL string, cdnURL string, expectedBucket string) string {
	parsedS3URL, err := url.Parse(s3URL)
	if err != nil {
		return s3URL
	}
	parsedCdnURL, err := url.Parse(cdnURL)
	if err != nil {
		return s3URL
	}
	s3URI, err := ParseURL(s3URL)
	if err != nil || s3URI.Bucket != expectedBucket {
		return s3URL
	}

	result := parsedCdnURL.Scheme + "://" + parsedCdnURL.Host + "/" + KeyEscape(s3URI.Key)
	if parsedS3URL.RawQuery != "" {
		result += "?" + parsedS3URL.RawQuery
	}
	if parsedS3URL.RawFragment != "" {
		result += "#" + parsedS3URL.RawFragment
	}

	return result
}

// ReplaceURLWithBucketMapping replaces an S3 URL with a CDN URL given a bucket-CDN map
// Similar with ReplaceURLIfBucketMatch but for multiple bucket and CDN
// Can be used together with envconfig, for instance:
// S3_CDN_MAPPING="core-ruangguru:https://imgix3.ruangguru.com,image-video-tracking:https://img-video-tracking.ruangguru.com"
func ReplaceURLWithBucketMapping(s3URL string, bucketCdnMap map[string]string) string {
	parsedS3URL, err := url.Parse(s3URL)
	if err != nil {
		return s3URL
	}
	s3URI, err := ParseURL(s3URL)
	if err != nil {
		return s3URL
	}
	if cdnURL, ok := bucketCdnMap[s3URI.Bucket]; ok {
		parsedCdnURL, err := url.Parse(cdnURL)
		if err != nil {
			return s3URL
		}

		result := parsedCdnURL.Scheme + "://" + parsedCdnURL.Host + "/" + KeyEscape(s3URI.Key)
		if parsedS3URL.RawQuery != "" {
			result += "?" + parsedS3URL.RawQuery
		}
		if parsedS3URL.RawFragment != "" {
			result += "#" + parsedS3URL.RawFragment
		}
		return result
	}
	return s3URL
}
