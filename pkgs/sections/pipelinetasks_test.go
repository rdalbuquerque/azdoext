package sections

import (
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/utils"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

func TestSortRecords(t *testing.T) {
	time1 := time.Date(2024, 1, 17, 20, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 17, 20, 30, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 17, 21, 0, 0, 0, time.UTC)
	records := []list.Item{
		listitems.PipelineRecordItem{
			StartTime: time1,
			Name:      "Job",
			Type:      "Job",
		},
		listitems.PipelineRecordItem{
			StartTime: time1,
			Name:      "Stage",
			Type:      "Stage",
		},
		listitems.PipelineRecordItem{
			StartTime: time2,
			Name:      "Task1",
		},
		listitems.PipelineRecordItem{
			StartTime: time3,
			Name:      "Task2",
		},
		listitems.PipelineRecordItem{
			Name: "Task2",
		},
	}
	sortedRecords := sortRecords(records)

	expectedSortedRecords := []list.Item{
		listitems.PipelineRecordItem{
			Name: "Task2",
		},
		listitems.PipelineRecordItem{
			StartTime: time1,
			Name:      "Stage",
			Type:      "Stage",
		},
		listitems.PipelineRecordItem{
			StartTime: time1,
			Name:      "Job",
			Type:      "Job",
		},
		listitems.PipelineRecordItem{
			StartTime: time2,
			Name:      "Task1",
		},
		listitems.PipelineRecordItem{
			StartTime: time3,
			Name:      "Task2",
		},
	}

	for i := 0; i < 100; i++ {
		for i, record := range sortedRecords {
			if record != expectedSortedRecords[i] {
				t.Errorf("Expected %v, got %v", expectedSortedRecords[i], record)
			}
		}
	}
}

func TestBuildPipelineRecordItem(t *testing.T) {
	sec := PipelineTasksSection{}
	st := time.Date(2024, 1, 17, 21, 0, 0, 0, time.UTC)
	node := &recordNode{
		Record: build.TimelineRecord{
			Name: utils.Ptr("Record 1"),
			Type: utils.Ptr("Task"),
			StartTime: &azuredevops.Time{
				Time: st,
			},
			State: &build.TimelineRecordStateValues.InProgress,
		},
	}
	record := sec.buildPipelineRecordItem(node)
	expectedRecord := listitems.PipelineRecordItem{
		Name:      "Record 1",
		Type:      "Task",
		StartTime: st,
		State:     build.TimelineRecordStateValues.InProgress,
		Result:    "",
	}
	if record != expectedRecord {
		t.Errorf("Expected %v, got %v", expectedRecord, record)
	}
}

func TestBuildPipelineRecordItemWithoutStartTime(t *testing.T) {
	sec := PipelineTasksSection{}
	node := &recordNode{
		Record: build.TimelineRecord{
			Name:  utils.Ptr("Record 1"),
			Type:  utils.Ptr("Task"),
			State: &build.TimelineRecordStateValues.InProgress,
		},
	}
	record := sec.buildPipelineRecordItem(node)
	expectedRecord := listitems.PipelineRecordItem{
		Name:      "Record 1",
		Type:      "Task",
		StartTime: time.Time{},
		State:     build.TimelineRecordStateValues.InProgress,
		Result:    "",
	}
	if record != expectedRecord {
		t.Errorf("Expected %v, got %v", expectedRecord, record)
	}
}
