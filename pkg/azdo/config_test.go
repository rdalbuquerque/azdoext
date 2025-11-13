package azdo

import "testing"

func TestGetOrgUrl(t *testing.T) {
	tests := []struct {
		name      string
		remoteUrl string
		want      string
	}{
		{
			name:      "HTTPS URL",
			remoteUrl: "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo",
			want:      "https://dev.azure.com/MyOrg",
		},
		{
			name:      "SSH URL",
			remoteUrl: "git@ssh.dev.azure.com:v3/MyOrg/MyProject/MyRepo",
			want:      "https://dev.azure.com/MyOrg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOrgUrl(tt.remoteUrl)
			if got != tt.want {
				t.Errorf("getOrgUrl(%q) = %q; want %q", tt.remoteUrl, got, tt.want)
			}
		})
	}
}

func TestGetOrgName(t *testing.T) {
	tests := []struct {
		name      string
		remoteUrl string
		want      string
	}{
		{
			name:      "HTTPS URL",
			remoteUrl: "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo",
			want:      "MyOrg",
		},
		{
			name:      "SSH URL",
			remoteUrl: "git@ssh.dev.azure.com:v3/MyOrg/MyProject/MyRepo",
			want:      "MyOrg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOrgName(tt.remoteUrl)
			if got != tt.want {
				t.Errorf("getOrgName(%q) = %q; want %q", tt.remoteUrl, got, tt.want)
			}
		})
	}
}

func TestGetProjectName(t *testing.T) {
	tests := []struct {
		name      string
		remoteUrl string
		want      string
	}{
		{
			name:      "HTTPS URL",
			remoteUrl: "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo",
			want:      "MyProject",
		},
		{
			name:      "SSH URL",
			remoteUrl: "git@ssh.dev.azure.com:v3/MyOrg/MyProject/MyRepo",
			want:      "MyProject",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getProjectName(tt.remoteUrl)
			if got != tt.want {
				t.Errorf("getProjectName(%q) = %q; want %q", tt.remoteUrl, got, tt.want)
			}
		})
	}
}

func TestGetRepositoryName(t *testing.T) {
	tests := []struct {
		name      string
		remoteUrl string
		want      string
	}{
		{
			name:      "HTTPS URL",
			remoteUrl: "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo",
			want:      "MyRepo",
		},
		{
			name:      "SSH URL",
			remoteUrl: "git@ssh.dev.azure.com:v3/MyOrg/MyProject/MyRepo",
			want:      "MyRepo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRepositoryName(tt.remoteUrl)
			if got != tt.want {
				t.Errorf("getRepositoryName(%q) = %q; want %q", tt.remoteUrl, got, tt.want)
			}
		})
	}
}
