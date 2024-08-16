package azdo

import "testing"

func TestGetOrgUrl(t *testing.T) {
	remoteUrl := "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo"
	want := "https://dev.azure.com/MyOrg"
	got := getOrgUrl(remoteUrl)
	if got != want {
		t.Errorf("getOrgUrl(%q) = %q; want %q", remoteUrl, got, want)
	}
}

func TestGetOrgName(t *testing.T) {
	remoteUrl := "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo"
	want := "MyOrg"
	got := getOrgName(remoteUrl)
	if got != want {
		t.Errorf("getOrgName(%q) = %q; want %q", remoteUrl, got, want)
	}
}

func TestGetProjectName(t *testing.T) {
	remoteUrl := "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo"
	want := "MyProject"
	got := getProjectName(remoteUrl)
	if got != want {
		t.Errorf("getProjectName(%q) = %q; want %q", remoteUrl, got, want)
	}
}

func TestGetRepositoryName(t *testing.T) {
	remoteUrl := "https://dev.azure.com/MyOrg/MyProject/_git/MyRepo"
	want := "MyRepo"
	got := getRepositoryName(remoteUrl)
	if got != want {
		t.Errorf("getRepositoryName(%q) = %q; want %q", remoteUrl, got, want)
	}
}
