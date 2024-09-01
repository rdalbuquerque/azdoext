package azdo

import (
	"context"
	"fmt"
	"io"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type LogClientInterface interface {
	GetTimelineRecordLog(context.Context, build.GetBuildLogArgs) (io.ReadCloser, error)
}

type LogClient struct {
	Client build.Client
}

func NewLogClient(ctx context.Context, orgurl, projectid, pat string) LogClientInterface {
	azdoconn := azuredevops.NewPatConnection(orgurl, pat)
	client, err := build.NewClient(ctx, azdoconn)
	if err != nil {
		panic(fmt.Sprintf("failed to create log client: %v", err))
	}
	return LogClient{
		Client: client,
	}
}

func (lc LogClient) GetTimelineRecordLog(ctx context.Context, args build.GetBuildLogArgs) (io.ReadCloser, error) {
	logReader, err := lc.Client.GetBuildLog(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline record logs: %w", err)
	}
	return logReader, nil
}
