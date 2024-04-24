package sections

import "azdoext/pkgs/listitems"

type OptionsStruct struct {
	OpenPR        listitems.OptionName
	GoToPipelines listitems.OptionName
	RunPipeline   listitems.OptionName
	GoToTasks     listitems.OptionName
}

var Options = OptionsStruct{
	OpenPR:        "Open PR",
	GoToPipelines: "Go to pipelines",
	RunPipeline:   "Run pipeline",
	GoToTasks:     "Go to tasks",
}
