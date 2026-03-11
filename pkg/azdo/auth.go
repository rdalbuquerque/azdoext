package azdo

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
)

// AuthProvider returns the Authorization header value ("Basic ..." or "Bearer ...").
type AuthProvider func(ctx context.Context) (string, error)

// azureDevOpsScope is the OAuth scope for Azure DevOps.
const azureDevOpsScope = "499b84ac-1321-427f-aa17-267ca6975798/.default"

// NewAuthProvider creates an AuthProvider using the following resolution order:
// 1. PAT (if AZDO_PERSONAL_ACCESS_TOKEN env var is set and valid)
// 2. Azure CLI credential (reuses az login tokens)
func NewAuthProvider(orgUrl string) (AuthProvider, error) {
	if pat := os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"); pat != "" {
		provider := newPatAuthProvider(pat)
		if validateAuthHeader(provider) {
			return provider, nil
		}
		fmt.Fprintln(os.Stderr, "PAT validation failed (expired or invalid), falling back to Azure CLI auth")
	}

	cliCred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		return nil, fmt.Errorf("no valid authentication found.\n\nEither:\n  - Run 'az login'\n  - Set AZDO_PERSONAL_ACCESS_TOKEN env var")
	}
	provider := newOAuthProvider(cliCred)
	if _, err := provider(context.Background()); err != nil {
		return nil, fmt.Errorf("Azure CLI credential failed.\n\nEither:\n  - Run 'az login'\n  - Set AZDO_PERSONAL_ACCESS_TOKEN env var")
	}
	return provider, nil
}

func newPatAuthProvider(pat string) AuthProvider {
	header := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(":"+pat)))
	return func(ctx context.Context) (string, error) {
		return header, nil
	}
}

func newOAuthProvider(cred azcore.TokenCredential) AuthProvider {
	return func(ctx context.Context) (string, error) {
		token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
			Scopes: []string{azureDevOpsScope},
		})
		if err != nil {
			return "", fmt.Errorf("failed to get OAuth token: %w", err)
		}
		return "Bearer " + token.Token, nil
	}
}

// validateAuthHeader checks if the auth header is valid by making a lightweight API call.
func validateAuthHeader(provider AuthProvider) bool {
	header, err := provider(context.Background())
	if err != nil {
		return false
	}
	req, err := http.NewRequest("GET", "https://app.vssps.visualstudio.com/_apis/profile/profiles/me?api-version=7.1", nil)
	if err != nil {
		return false
	}
	req.Header.Add("Authorization", header)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// NewConnection creates an azuredevops.Connection using the given auth header.
func NewConnection(orgUrl string, authHeader string) *azuredevops.Connection {
	return &azuredevops.Connection{
		AuthorizationString:     authHeader,
		BaseUrl:                 orgUrl,
		SuppressFedAuthRedirect: true,
	}
}
