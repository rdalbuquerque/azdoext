package azdo

import (
	"context"
	"fmt"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type ErrNoBuildsFound struct{}

func (e ErrNoBuildsFound) Error() string {
	return "no builds found"
}

type BuildClientInterface interface {
	GetBuildTimelineRecords(context.Context, build.GetBuildTimelineArgs) ([]build.TimelineRecord, error)
	QueueBuild(context.Context, build.QueueBuildArgs) (int, error)
	GetDefinitions(context.Context, build.GetDefinitionsArgs) ([]build.BuildDefinitionReference, error)
	GetBuilds(context.Context, build.GetBuildsArgs) ([]build.Build, error)
}

type BuildClient struct {
	build.Client
	projectid string
}

func NewBuildClient(ctx context.Context, orgurl, projectid, pat string) BuildClientInterface {
	azdoconn := azuredevops.NewPatConnection(orgurl, pat)
	client, err := build.NewClient(ctx, azdoconn)
	if err != nil {
		panic(fmt.Sprintf("failed to create build client: %v", err))
	}
	return BuildClient{
		projectid: projectid,
		Client:    client,
	}
}

func (b BuildClient) GetBuildTimelineRecords(ctx context.Context, args build.GetBuildTimelineArgs) ([]build.TimelineRecord, error) {
	args.Project = &b.projectid
	timeline, err := b.Client.GetBuildTimeline(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get build timeline records: %w", err)
	}
	return *timeline.Records, nil
}

func (b BuildClient) QueueBuild(ctx context.Context, args build.QueueBuildArgs) (int, error) {
	args.Project = &b.projectid
	build, err := b.Client.QueueBuild(ctx, args)
	if err != nil {
		return 0, fmt.Errorf("failed to queue build: %w", err)
	}
	return *build.Id, nil
}

func (b BuildClient) GetDefinitions(ctx context.Context, args build.GetDefinitionsArgs) ([]build.BuildDefinitionReference, error) {
	args.Project = &b.projectid
	defs, err := b.Client.GetDefinitions(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get build definitions: %w", err)
	}
	definitions := defs.Value
	if len(definitions) == 0 {
		return nil, ErrNoBuildsFound{}
	}
	return definitions, nil
}

func (b BuildClient) GetBuilds(ctx context.Context, args build.GetBuildsArgs) ([]build.Build, error) {
	args.Project = &b.projectid
	buildsResponse, err := b.Client.GetBuilds(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get builds: %w", err)
	}
	builds := buildsResponse.Value
	if len(builds) == 0 {
		return nil, ErrNoBuildsFound{}
	}
	return builds, nil
}
