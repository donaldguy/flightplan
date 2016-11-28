package flightplan

import (
	"github.com/concourse/atc"
	configParser "github.com/concourse/atc/config"
)

type JobName string
type ResourceName string

type ResourcesByType struct {
	AllInputsOfJob        map[JobName][]ResourceName
	Entrypoints           map[ResourceName][]Entrypoint
	MidtriggersByResource map[ResourceName][]*Midtrigger
	Byproducts            map[JobName][]ResourceName
	Products              map[ResourceName]JobName
}

type Entrypoint struct {
	ResourceName
	TriggeredJob JobName
}

type Midtrigger struct {
	ResourceName
	Passed       []JobName
	TriggeredJob JobName
}

func resourcesByType(config *atc.Config) *ResourcesByType {
	resources := &ResourcesByType{}
	resources.fillInputs(config)
	resources.fillOutputs(config)
	return resources
}

// classify inputs as either:
//  entrypoints:             input -> job
//  midtriggers: [passed] -> input -> job
func (r *ResourcesByType) fillInputs(config *atc.Config) {
	r.Entrypoints = make(map[ResourceName][]Entrypoint)
	r.AllInputsOfJob = make(map[JobName][]ResourceName)
	r.MidtriggersByResource = make(map[ResourceName][]*Midtrigger)

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
						r.Entrypoints[inputName] = []Entrypoint{}
					}
					r.Entrypoints[inputName] = append(
						r.Entrypoints[inputName],
						Entrypoint{inputName, jobName},
					)
				} else {
					// it is a middle stage trigger

					mt := &Midtrigger{
						ResourceName: inputName,
						Passed:       make([]JobName, len(input.Passed)),
						TriggeredJob: jobName,
					}
					for i, name := range input.Passed {
						mt.Passed[i] = JobName(name)
					}

					if _, ok := r.MidtriggersByResource[inputName]; !ok {
						r.MidtriggersByResource[inputName] = []*Midtrigger{}
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

func (r *ResourcesByType) fillOutputs(config *atc.Config) {
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
