package sections

import (
	"azdoext/pkg/listitems"
	"azdoext/pkg/utils"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
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
	Children []*RecordNode
}

type RootNode struct {
	Roots []*RecordNode
}

func (r RootNode) TimelineRecords() []build.TimelineRecord {
	var toTimelineRecords func([]build.TimelineRecord, []*RecordNode) []build.TimelineRecord
	toTimelineRecords = func(records []build.TimelineRecord, nodes []*RecordNode) []build.TimelineRecord {
		for _, node := range nodes {
			records = append(records, node.Record)
			records = toTimelineRecords(records, node.Children)
		}
		return records
	}

	tlrecords := []build.TimelineRecord{}
	tlrecords = toTimelineRecords(tlrecords, r.Roots)
	return tlrecords
}

func sortRecordTreeByOrder(nodes []*RecordNode) {
	slices.SortFunc(nodes, func(n1, n2 *RecordNode) int {
		return cmp.Compare(*n1.Record.Order, *n2.Record.Order)
	})
	for _, node := range nodes {
		sortRecordTreeByOrder(node.Children)
	}
}

func printRecords(nodes []*RecordNode) {
	for _, node := range nodes {
		fmt.Println(strings.Join([]string{
			strconv.Itoa(*(*node).Record.Order),
			*(*node).Record.Type,
			strings.ReplaceAll(*(*node).Record.Name, " ", ""),
			(*(*node).Record.Id).String(),
		}, "_"))
		printRecords(node.Children)
	}
}

func (r *RootNode) Print() {
	rjson, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatalf("unable to marshal root node: %v", err)
	}
	fmt.Println(string(rjson))
}

func sortRecords(records []build.TimelineRecord) []build.TimelineRecord {
	// build a hashmap so each record is easily accessible
	recordtree := make(map[uuid.UUID]*RecordNode)
	for _, record := range records {
		recordtree[*record.Id] = &RecordNode{Record: record}
	}
	// build a tree so the hierarchy Stage->Phase->Job->Task is respected
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
	// The siblings are in a random order, so sort each Children slice
	sortRecordTreeByOrder(root.Roots)

	sorted := root.TimelineRecords()

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
	fmt.Println()
	fmt.Println("sorted timeline records:")
	for _, record := range sortedRecords {
		fmt.Println(strings.Join([]string{
			strconv.Itoa(*record.Order),
			*record.Type,
			strings.ReplaceAll(*record.Name, " ", ""),
			(*record.Id).String(),
		}, "_"))
	}
	sortedRecordItems := convertToItems(sortedRecords)

	expectedOrder := []uuid.UUID{
		uuid.MustParse("8b66ed65-25bc-5280-3c7b-27a8e21abd86"), // Stage "dev"
		uuid.MustParse("a2bc865c-88d2-5fb4-6ae3-0a9e58465cba"), // Phase "dev1"
		uuid.MustParse("4c7d219d-95bf-57d0-2917-329a599b9ffb"), // Job "dev1"
		uuid.MustParse("ed11269f-deae-4788-8c56-3093f8ea2428"), // Task "Initialize job" (order 1, dev1)
		uuid.MustParse("69b2271d-7067-5aab-ded0-475817697011"), // Task "Checkout test azdoext@main to s" (order 2, dev1)
		uuid.MustParse("773b0d8c-32d8-52ab-6631-ef8a159e835e"), // Task "Run a one‐line script" (order 3, dev1)
		uuid.MustParse("c36b0d38-5d46-5c27-0211-1ea75300db80"), // Task "Run a multi‐line script in dev1" (order 4, dev1)
		uuid.MustParse("3989503b-981a-5c11-635b-185065ee887c"), // Task "Run another multi‐line script in dev1" (order 5, dev1)
		uuid.MustParse("b029ea14-e7b2-4f1e-ae84-5eb72e29ffa2"), // Task "Post‐job: Checkout test azdoext@main to s" (order 6, dev1)
		uuid.MustParse("03488951-7a5e-4881-9cc6-267a3e2fd20f"), // Task "Finalize Job" (order 7, dev1)

		uuid.MustParse("8b5e9ccb-a181-50bb-762c-540add245a8d"), // Phase "dev2"
		uuid.MustParse("64871a50-40ad-53c9-977c-96af11b04bdd"), // Job "dev2"
		uuid.MustParse("c385331c-57da-469e-9f15-0165feb4feef"), // Task "Initialize job" (order 1, dev2)
		uuid.MustParse("d40608a3-c849-5b8c-9d71-54b86d7461ec"), // Task "Checkout test azdoext@main to s" (order 2, dev2)
		uuid.MustParse("e79c6973-6966-51be-c29d-503f1cacc7e0"), // Task "Run a one‐line script2" (order 3, dev2)
		uuid.MustParse("06c089d8-ccf0-5b18-d72b-dc59f3e3067c"), // Task "Run a multi‐line script in dev2" (order 4, dev2)
		uuid.MustParse("0834e1ec-4a9f-5d2e-447d-5a3ab61ed58f"), // Task "Run another multi‐line script in dev2" (order 5, dev2)
		uuid.MustParse("c790f8e1-ba9d-4fbd-8e21-dc4fb83a737e"), // Task "Post‐job: Checkout test azdoext@main to s" (order 6, dev2)
		uuid.MustParse("a0d59018-1563-4d8b-8867-b8b5d9f40e9a"), // Task "Finalize Job" (order 7, dev2)

		uuid.MustParse("6263517a-7425-5966-a12a-e8a34e96b275"), // Stage "stg"
		uuid.MustParse("6a94b20b-512b-500e-08ed-1d7c440898fc"), // Phase "stg1"
		uuid.MustParse("a5297c70-ab47-5725-2f5f-7fe22b600b1e"), // Job "stg1"
		uuid.MustParse("ae34bc12-7d42-40cb-a8a8-3c767cf504f2"), // Task "Initialize job" (order 1, stg1)
		uuid.MustParse("e919959e-5f3c-5c0c-fa5d-4ce77f0d77f0"), // Task "Checkout test azdoext@main to s" (order 2, stg1)
		uuid.MustParse("4e980a5d-7691-5f50-d8b8-7bb339216e99"), // Task "Run a one‐line script" (order 3, stg1)
		uuid.MustParse("e8fc54e6-7623-5324-dbe2-a15d1f75cee2"), // Task "Run a multi‐line script in stg1" (order 4, stg1; placeholder if missing)
		uuid.MustParse("9243b30b-d7bb-5176-0f39-40575ad57037"), // Task "Run another multi‐line script in stg1" (order 5, stg1)
		uuid.MustParse("37887a64-c743-4117-94a5-b04ff9113bd8"), // Task "Post‐job: Checkout test azdoext@main to s" (order 6, stg1)
		uuid.MustParse("7582a0c4-ce44-4af0-bab4-f4e013cbbf94"), // Task "Finalize Job" (order 7, stg1)

		uuid.MustParse("95b4af66-779c-5659-3b99-6289447b8b30"), // Phase "stg2"
		uuid.MustParse("1aea2d54-bf70-5590-863b-cd99f4b254f6"), // Job "stg2"
		uuid.MustParse("de7c0fb0-ab75-4f58-8b97-4de27facfed5"), // Task "Initialize job" (order 1, stg2)
		uuid.MustParse("22405931-0591-52f7-b3f1-c49821e3819c"), // Task "Checkout test azdoext@main to s" (order 2, stg2)
		uuid.MustParse("dec1d546-59c5-5c58-e841-c2c47e81b951"), // Task "Run a one‐line script" (order 3, stg2)
		uuid.MustParse("a24b5576-4963-57e9-a06f-5badfc0604d0"), // Task "Run a multi‐line script in stg2" (order 4, stg2)
		uuid.MustParse("45fab9b0-8d48-5255-a1a4-a23eec979e1f"), // Task "Run another multi‐line script in stg2" (order 5, stg2)
		uuid.MustParse("99c2b995-a2d7-4da8-8d50-8125a2d90402"), // Task "Post‐job: Checkout test azdoext@main to s" (order 6, stg2)
		uuid.MustParse("705b900a-26b7-44dc-baff-ea7cded19645"), // Task "Finalize Job" (order 7, stg2)

		uuid.MustParse("acf23ef4-cae6-4180-a610-e6887e184837"), // Job "Finalize build"
		uuid.MustParse("78db2542-c627-4140-8a7a-d06178fff4e4"), // Task "Report build status" (order 2147483647)
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
