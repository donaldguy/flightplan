package flightplan

import (
	"github.com/concourse/atc"
	configParser "github.com/concourse/atc/config"
)

// JobName is a string representing the name of a job
type JobName string

// ResourceName is a string representing the name of a resource
type ResourceName string

type resourcesByType struct {
	AllInputsOfJob        map[JobName][]ResourceName
	Entrypoints           map[ResourceName][]entrypoint
	MidtriggersByResource map[ResourceName][]*midtrigger
	Byproducts            map[JobName][]ResourceName
	Products              map[ResourceName]JobName
}

type entrypoint struct {
	ResourceName
	TriggeredJob JobName
}

type midtrigger struct {
	ResourceName
	Passed       []JobName
	TriggeredJob JobName
}

func newResourcesByType(config *atc.Config) *resourcesByType {
	resources := &resourcesByType{}
	resources.fillInputs(config)
	resources.fillOutputs(config)
	return resources
}

// classify inputs as either:
//  entrypoints:             input -> job
//  midtriggers: [passed] -> input -> job
func (r *resourcesByType) fillInputs(config *atc.Config) {
	r.Entrypoints = make(map[ResourceName][]entrypoint)
	r.AllInputsOfJob = make(map[JobName][]ResourceName)
	r.MidtriggersByResource = make(map[ResourceName][]*midtrigger)

	for _, job := range config.Jobs {
		jobName := JobName(job.Name)
		inputs := configParser.JobInputs(job)
		r.AllInputsOfJob[jobName] = make([]ResourceName, len(inputs))

		for i, input := range inputs {
			inputName := ResourceName(input.Resource)
			r.AllInputsOfJob[jobName][i] = inputName
			if input.Trigger {
				// is an entrypoint, record as such
				if len(input.Passed) == 0 {
					if _, ok := r.Entrypoints[inputName]; !ok {
						r.Entrypoints[inputName] = []entrypoint{}
					}
					r.Entrypoints[inputName] = append(
						r.Entrypoints[inputName],
						entrypoint{inputName, jobName},
					)
				} else {
					// it is a middle stage trigger

					mt := &midtrigger{
						ResourceName: inputName,
						Passed:       make([]JobName, len(input.Passed)),
						TriggeredJob: jobName,
					}
					for i, name := range input.Passed {
						mt.Passed[i] = JobName(name)
					}

					if _, ok := r.MidtriggersByResource[inputName]; !ok {
						r.MidtriggersByResource[inputName] = []*midtrigger{}
					}
					r.MidtriggersByResource[inputName] = append(
						r.MidtriggersByResource[inputName],
						mt,
					)
				}
			}
		}
	}
}

func (r *resourcesByType) fillOutputs(config *atc.Config) {
	r.Products = make(map[ResourceName]JobName)
	r.Byproducts = make(map[JobName][]ResourceName)

	for _, job := range config.Jobs {
		jobName := JobName(job.Name)
		for _, output := range configParser.JobOutputs(job) {
			outputName := ResourceName(output.Resource)

			//XXX: this definition is insufficient for a solely name-based identity
			//  it could be augmented by switching to a (at least) name+passed id
			//  however it is good enough for the pipelines I'm currently looking at
			if _, isMiddtrigger := r.MidtriggersByResource[outputName]; !isMiddtrigger {
				r.Products[outputName] = jobName
			}

			if _, ok := r.Byproducts[jobName]; !ok {
				r.Byproducts[jobName] = []ResourceName{}
			}
			r.Byproducts[jobName] = append(r.Byproducts[jobName], outputName)
		}
	}
}
