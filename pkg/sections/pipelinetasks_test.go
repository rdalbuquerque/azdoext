package sections

import (
	"azdoext/pkg/listitems"
	"azdoext/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

//type BuildClientInterface interface {
//	GetBuildTimelineRecords(context.Context, build.GetBuildTimelineArgs) ([]build.TimelineRecord, error)
//	GetFilteredBuildTimelineRecords(context.Context, build.GetBuildTimelineArgs) ([]build.TimelineRecord, error)
//	GetTimelineRecordLog(context.Context, build.GetBuildLogArgs) (io.ReadCloser, error)
//	QueueBuild(context.Context, build.QueueBuildArgs) (int, error)
//	GetDefinitions(context.Context, build.GetDefinitionsArgs) ([]build.BuildDefinitionReference, error)
//	GetBuilds(context.Context, build.GetBuildsArgs) ([]build.Build, error)
//

type buildClient struct{}

func (b buildClient) GetBuildTimelineRecords(ctx context.Context, args build.GetBuildTimelineArgs) ([]build.TimelineRecord, error) {
	return []build.TimelineRecord{}, nil
}

func (b buildClient) GetFilteredBuildTimelineRecords(ctx context.Context, args build.GetBuildTimelineArgs) ([]build.TimelineRecord, error) {
	recordsBytes, err := os.ReadFile("parallel-stages4.json")
	if err != nil {
		log.Fatalf("unable to read record test file: %v", err)
	}
	var records []build.TimelineRecord
	json.Unmarshal(recordsBytes, &records)
	return slices.DeleteFunc(records, func(record build.TimelineRecord) bool {
		return !slices.Contains([]string{"Stage", "Job", "Task", "Phase"}, *record.Type)
	}), nil
}
func (b buildClient) GetTimelineRecordLog(ctx context.Context, args build.GetBuildLogArgs) (io.ReadCloser, error) {
	return nil, nil
}

func (b buildClient) QueueBuild(ctx context.Context, args build.QueueBuildArgs) (int, error) {
	return 0, nil
}

func (b buildClient) GetDefinitions(ctx context.Context, args build.GetDefinitionsArgs) ([]build.BuildDefinitionReference, error) {
	return []build.BuildDefinitionReference{}, nil
}

func (b buildClient) GetBuilds(ctx context.Context, args build.GetBuildsArgs) ([]build.Build, error) {
	return []build.Build{}, nil
}

type RecordNode struct {
	Record   build.TimelineRecord
	Children []RecordNode
}

type RootNode struct {
	Roots []RecordNode
}

func print(nodes []RecordNode) {
	for _, node := range nodes {
		fmt.Printf("type: %s | order: %d | name: %s", *node.Record.Type, *node.Record.Order, *node.Record.Name)
		print(node.Children)
	}
}

func (r RootNode) Print() {
	rjson, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatalf("unable to marshal root node: %v", err)
	}
	fmt.Println(string(rjson))
	// for _, node := range r.Roots {
	// 	print(node.Children)
	// }
}

func sortRecords(records []build.TimelineRecord) []build.TimelineRecord {
	sorted := []build.TimelineRecord{}

	recordtree := make(map[uuid.UUID]RecordNode)
	for _, record := range records {
		recordtree[*record.Id] = RecordNode{Record: record}
	}

	root := RootNode{}
	for _, record := range records {
		if record.ParentId != nil {
			node := recordtree[*record.ParentId]
			node.Children = append(node.Children, recordtree[*record.Id])
			recordtree[*record.ParentId] = node
		} else {
			root.Roots = append(root.Roots, recordtree[*record.Id])
		}
	}
	root.Print()

	return sorted
}

func convertToItems(records []build.TimelineRecord) []listitems.PipelineRecordItem {
	recorditems := []listitems.PipelineRecordItem{}
	for _, record := range records {
		recorditems = append(recorditems, buildPipelineRecordItemFromRecord(record))
	}
	return recorditems
}

func TestSortRecords(t *testing.T) {
	records, err := buildClient{}.GetFilteredBuildTimelineRecords(context.Background(), build.GetBuildTimelineArgs{})
	if err != nil {
		t.Fatalf("unable to get filtered records: %v", err)
	}

	sortedRecords := sortRecords(records)
	sortedRecordItems := convertToItems(sortedRecords)

	expectedOrder := []uuid.UUID{
		uuid.MustParse("8b66ed65-25bc-5280-3c7b-27a8e21abd86"),
		uuid.MustParse("cd98967c-668a-5f0d-6f6c-b174f63c7461"),
		uuid.MustParse("cdab10ca-a881-54d9-8032-f83c591845fb"),
		uuid.MustParse("53ffaefa-cef2-47b5-b593-de92433c94ff"),
		uuid.MustParse("a2af6525-8e3c-5677-a39d-4e4385f1451f"),
		uuid.MustParse("fe5b427e-14c7-5585-1204-236ce8326bf1"),
		uuid.MustParse("55ac1c55-a885-52f6-df31-03dc3350ef27"),
		uuid.MustParse("9485b111-bccb-56ca-e536-84d3407ba305"),
		uuid.MustParse("d6170369-14e1-4c80-85f8-9508b0f9f90b"),
		uuid.MustParse("ec334bd8-76fa-400f-a2db-504991354d7b"),
		uuid.MustParse("6263517a-7425-5966-a12a-e8a34e96b275"),
		uuid.MustParse("b7e0a3a7-238f-51c3-71d8-9547ee243767"),
		uuid.MustParse("d5a6aa2d-6ad8-50b1-8d63-f54f233c6dd4"),
		uuid.MustParse("c058c938-17fa-459d-89a5-eb611c911571"),
		uuid.MustParse("687e181f-e66f-5128-3476-cd0cd4ae68b5"),
		uuid.MustParse("c5b80bca-141a-5f54-acee-64c2f6a09f05"),
		uuid.MustParse("806335ed-c811-5d63-0bd0-d6bddfb7a6e9"),
		uuid.MustParse("d4f75645-8eef-58e1-a4d0-23657797fea1"),
		uuid.MustParse("faabe6ac-f295-4825-bc72-f61091cda074"),
		uuid.MustParse("fea117c0-ff33-4986-8a62-bb9292a717C7"),
		uuid.MustParse("acf23ef4-cae6-4180-a610-e6887e184837"),
		uuid.MustParse("78db2542-c627-4140-8a7a-d06178fff4e4"),
	}

	if len(sortedRecordItems) != len(expectedOrder) {
		t.Fatalf("expected len: %d, got length: %d", len(expectedOrder), len(sortedRecordItems))
	}

	for i, record := range sortedRecordItems {
		pipelineRecordItem := record
		if pipelineRecordItem.Id != expectedOrder[i] {
			t.Errorf("Expected %s, got %s", expectedOrder[i], pipelineRecordItem.Id)
		}
	}

}

func TestBuildPipelineRecordItemWithoutStartTime(t *testing.T) {
	id := utils.Ptr(uuid.New())
	node := Node{
		Record: build.TimelineRecord{
			Name:  utils.Ptr("Record 1"),
			Type:  utils.Ptr("Task"),
			State: &build.TimelineRecordStateValues.InProgress,
			Id:    id,
		},
	}
	record := buildPipelineRecordItem(node)
	expectedRecord := listitems.PipelineRecordItem{
		Name:      "Record 1",
		Type:      "Task",
		StartTime: time.Time{},
		State:     build.TimelineRecordStateValues.InProgress,
		Result:    "",
		Id:        *id,
	}
	if record != expectedRecord {
		t.Errorf("Expected %v, got %v", expectedRecord, record)
	}
}
