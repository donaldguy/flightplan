package flightplan

type ResourceNode struct {
	Name ResourceName

	OutputBy *JobNode
	Passed   []JobName

	TriggeredJobs []*JobNode
}

type JobNode struct {
	Name        JobName
	TriggeredBy *ResourceNode
	AlsoNeeds   []ResourceName

	Outputs []*ResourceNode
}

func (p *Pipeline) GraphStartingFrom(resourceName ResourceName) *ResourceNode {
	g := &ResourceNode{
		Name:          resourceName,
		TriggeredJobs: []*JobNode{},
		OutputBy:      nil,
	}

	g.resolveIn(p)
	//TODO: intelligentlly de-duplicate identical paths on a multi-resource
	// fan-in -_-
	// sketch: look for resources that have the same (recursive?) OutputBy
	//         that share a TriggeredJob ?

	return g
}

func (r *ResourceNode) resolveIn(p *Pipeline) {
	if _, done := p.Products[r.Name]; done {
		if r.OutputBy != nil {
			return
		}
	}
	for _, entrypoint := range p.Entrypoints[r.Name] {
		j := &JobNode{
			Name:        entrypoint.TriggeredJob,
			TriggeredBy: r,
		}
		r.TriggeredJobs = append(r.TriggeredJobs, j)
		j.resolveIn(p)
	}

	midtriggers := p.MidtriggersByResource[r.Name]
	// loop through the midtrigger nodes and look for outs whose passed set
	// includes a job we entrypointed, if any. For these we will create a
	// "passthrough" resource annotated with the passed set, akin to the
	// "light-text" nodes displayed in a the concourse web UI graph
	mtPassesThrough := make(map[*Midtrigger]*JobNode, len(midtriggers))
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

	for mt, tj := range mtPassesThrough {
		if tj != nil {
			sytheticR := &ResourceNode{
				Name:     r.Name,
				Passed:   mt.Passed[:],
				OutputBy: tj,
			}
			tj.Outputs = append(tj.Outputs, sytheticR)
			j := &JobNode{
				Name:        mt.TriggeredJob,
				TriggeredBy: sytheticR,
				Outputs:     []*ResourceNode{},
			}
			sytheticR.TriggeredJobs = []*JobNode{j}
			j.resolveIn(p)
		} else {
			j := &JobNode{
				Name:        mt.TriggeredJob,
				TriggeredBy: r,
				Outputs:     []*ResourceNode{},
			}
			r.TriggeredJobs = append(r.TriggeredJobs, j)
			j.resolveIn(p)
		}
	}
}

func (j *JobNode) resolveIn(p *Pipeline) {
	for _, rName := range p.Byproducts[j.Name] {
		r := &ResourceNode{
			Name:          rName,
			TriggeredJobs: []*JobNode{},
			OutputBy:      j,
		}
		j.Outputs = append(j.Outputs, r)
		r.resolveIn(p)
	}

	j.AlsoNeeds = make([]ResourceName, len(p.AllInputsOfJob[j.Name])-1)
	i := 0
	for _, input := range p.AllInputsOfJob[j.Name] {
		if input != j.TriggeredBy.Name {
			j.AlsoNeeds[i] = input
			i++
		}
	}
}
