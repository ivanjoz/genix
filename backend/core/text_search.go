package core

import (
	"app/db"
	"context"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ivanjoz/genix-orm/search/daemon"
	"github.com/ivanjoz/genix-orm/search/genixs3"
)

// ConfigureTextSearch installs the ORM's text-search backend from GENIXSEARCH_URL, before any
// write to a table with TextSearch:
//
//   - "s3express://bucket/prefix" or "s3://bucket/prefix": genix-s3, the index in that bucket
//     under prefix ("genix/" when empty). An S3 Express One Zone directory bucket
//     ("name--use1-az4--x-s3") is the fast choice; it must be in the region of GetAwsConfig.
//   - anything else: the GenixSearch daemon over TCP (see ParseGenixSearchURL), which needs
//     GENIXSEARCH_PASSWORD in prod.
//
// The ORM packages cannot import core (cycle), so the resolved config is pushed in from here.
func ConfigureTextSearch() {
	if bucket, root, ok := parseGenixS3URL(Env.GENIXSEARCH_URL); ok {
		// Compaction defaults to CompactAuto: inside the write on Lambda, in a goroutine on a server.
		backend, err := genixs3.New(context.Background(), genixs3.Config{
			Bucket: bucket,
			Root:   root,
			S3:     s3.NewFromConfig(GetAwsConfig()),
			OnCompactError: func(table, partition string, err error) {
				Log("text search: compaction of", table, partition, "failed:", err)
			},
		})
		if err != nil {
			panic("invalid genix-s3 text search configuration: " + err.Error())
		}
		db.SetTextSearchBackend(backend)
		return
	}

	host, port := ParseGenixSearchURL(Env.GENIXSEARCH_URL)
	password := strings.TrimSpace(Env.GENIXSEARCH_PASSWORD)
	if password == "" && Env.IS_PROD {
		Log("text search: GENIXSEARCH_PASSWORD empty in prod; daemon writes will fail at handshake")
	}
	daemon.Configure(host, port, password)
	db.SetTextSearchBackend(daemon.NewBackend())
}

// parseGenixS3URL splits "s3express://bucket/prefix" and "s3://bucket/prefix". ok is false for
// any other scheme.
func parseGenixS3URL(raw string) (bucket, root string, ok bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "s3" && parsed.Scheme != "s3express") || parsed.Host == "" {
		return "", "", false
	}

	root = strings.Trim(parsed.Path, "/")
	if root != "" {
		root += "/"
	}
	return parsed.Host, root, true
}
