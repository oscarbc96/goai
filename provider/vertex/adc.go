package vertex

import (
	"context"
	"time"

	"golang.org/x/oauth2/google"

	"github.com/zendev-sh/goai/provider"
)

// ADCTokenSource creates a TokenSource that uses Google Application Default Credentials.
//
// It auto-detects credentials from (in order):
//  1. GOOGLE_APPLICATION_CREDENTIALS env var (service account JSON file)
//  2. gcloud CLI credentials (~/.config/gcloud/application_default_credentials.json)
//  3. GCE metadata service (when running on Google Cloud)
//
// This is the Go equivalent of Vercel's google-auth-library auto-detection.
//
// Usage:
//
//	model := vertex.Chat("gemini-2.5-pro",
//		vertex.WithTokenSource(vertex.ADCTokenSource()),
//		vertex.WithProject("my-project"),
//	)
func ADCTokenSource() provider.TokenSource {
	return provider.CachedTokenSource(func(ctx context.Context) (*provider.Token, error) {
		creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, err
		}
		tok, err := creds.TokenSource.Token()
		if err != nil {
			return nil, err
		}
		return &provider.Token{
			Value:     tok.AccessToken,
			ExpiresAt: tok.Expiry.Add(-30 * time.Second), // refresh 30s early
		}, nil
	})
}
