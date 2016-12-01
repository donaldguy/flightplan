package flightplan

type Graph struct {
	Start    *ResourceNode
	JobIndex map[JobName]*JobNode
	pipeline *Pipeline
}

//ResourceNode is a node in a linked structure representing an instance of a resource
// in a to-be-run concourse pipeline
type ResourceNode struct {
	Name ResourceName

	OutputBy *JobNode
	Passed   []JobName

	TriggeredJobs []*JobNode
}

//JobNode is a node in a linked structure representing a job in a to-be-run concourse pipeline
type JobNode struct {
	Name        JobName
	TriggeredBy []*ResourceNode
	AlsoNeeds   []ResourceName

	Outputs []*ResourceNode
}

// GraphStartingFrom returns a
func (p *Pipeline) GraphStartingFrom(resourceName string) *Graph {
	graph := &Graph{
		Start: &ResourceNode{
			Name:          ResourceName(resourceName),
			TriggeredJobs: []*JobNode{},
			OutputBy:      nil,
		},
		JobIndex: make(map[JobName]*JobNode),
		pipeline: p,
	}

	graph.resolveResource(graph.Start)

	return graph
}

func (graph *Graph) resolveResource(r *ResourceNode) {
	if _, done := graph.pipeline.resourcesByType.Products[r.Name]; done {
		if r.OutputBy != nil {
			return
		}
	}
	for _, entrypoint := range graph.pipeline.resourcesByType.Entrypoints[r.Name] {
		j := &JobNode{
			Name:        entrypoint.TriggeredJob,
			TriggeredBy: []*ResourceNode{r},
			Outputs:     []*ResourceNode{},
		}
		j = graph.resolveJob(j)
		r.TriggeredJobs = append(r.TriggeredJobs, j)
	}

	midtriggers := graph.pipeline.resourcesByType.MidtriggersByResource[r.Name]
	// loop through the midtrigger nodes and look for outs whose passed set
	// includes a job we entrypointed, if any. For these we will create a
	// "passthrough" resource annotated with the passed set, akin to the
	// "light-text" nodes displayed in a the concourse web UI graph
	//
	// FIXME: this only works cause we (at tulip) happen to only do single-stage
	// passthrough - thus the chain goes immediately through an entrypoint
	// ; for a longer chain, we need to do an iterative resolution process filling
	// "left to right" across jobs with later stage dependencies, until we've populated
	// the whole job index(?) as len(graph.resourcesByType.AllInputsOfJob) == len(graph.JobIndex)
	// this is probably a process closer to how concourse actually works
	mtPassesThrough := make(map[*midtrigger]*JobNode, len(midtriggers))
	for _, mt := range midtriggers {
		mtPassesThrough[mt] = nil
		for _, tj := range r.TriggeredJobs {
			for _, passedJob := range mt.Passed {
				if tj.Name == passedJob {
					mtPassesThrough[mt] = tj
				}
			}
		}
	}

	for mt, jobPassedThrough := range mtPassesThrough {
		if jobPassedThrough != nil {
			//we construct a "shadow resource" that represents the resource after
			//it has passed the job it passes through.
			// then we string
			// (jobPassedThrough) -> (shadow resource) -> (triggered job)

			shadowR := &ResourceNode{
				Name:     r.Name,
				Passed:   mt.Passed[:],
				OutputBy: jobPassedThrough,
			}
			jobPassedThrough.Outputs = append(jobPassedThrough.Outputs, shadowR)
			j := &JobNode{
				Name:        mt.TriggeredJob,
				TriggeredBy: []*ResourceNode{shadowR},
				Outputs:     []*ResourceNode{},
			}
			j = graph.resolveJob(j)
			shadowR.TriggeredJobs = append(shadowR.TriggeredJobs, j)
		} else {
			j := &JobNode{
				Name:        mt.TriggeredJob,
				TriggeredBy: []*ResourceNode{r},
				Outputs:     []*ResourceNode{},
			}
			j = graph.resolveJob(j)
			r.TriggeredJobs = append(r.TriggeredJobs, j)
		}
	}
}

func (graph *Graph) resolveJob(jin *JobNode) *JobNode {
	var j *JobNode
	var alreadyExisted bool

	if j, alreadyExisted = graph.JobIndex[jin.Name]; alreadyExisted {
		j.TriggeredBy = append(j.TriggeredBy, jin.TriggeredBy...)
	} else {
		//we only want to calculate our outputs once, they shouldn't change
		j = jin
		graph.JobIndex[j.Name] = j
		for _, rName := range graph.pipeline.resourcesByType.Byproducts[j.Name] {
			r := &ResourceNode{
				Name:          rName,
				TriggeredJobs: []*JobNode{},
				OutputBy:      j,
			}
			//r = graph.registeredResource(r)

			//only products need to be added. The rest is handled by Midtrigger Resoultion
			// if _, ok := graph.pipeline.resourcesByType.Products[rName]; ok {

			//}

			graph.resolveResource(r) //although with a sane nil case, this might help with the above FIXME
			j.Outputs = append(j.Outputs, r)
		}
	}

	j.AlsoNeeds = make([]ResourceName, len(graph.pipeline.resourcesByType.AllInputsOfJob[j.Name])-len(j.TriggeredBy))
	i := 0
OUTER:
	for _, input := range graph.pipeline.resourcesByType.AllInputsOfJob[j.Name] {
		for _, trigger := range j.TriggeredBy {
			if input == trigger.Name {
				continue OUTER
			}
		}
		j.AlsoNeeds[i] = input
		i++
	}
	return j
}
