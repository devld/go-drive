package scheduled

import "fmt"

var registeredJobs  = make(map[string]*JobDefinition)

func RegisterJob(job JobDefinition) {
	if _, exists := registeredJobs[job.Name]; exists {
		panic(fmt.Sprintf("job '%s' already registered", job.Name))
	}
	registeredJobs[job.Name] = &job
}

func GetJob(name string) *JobDefinition {
	jd, exists := registeredJobs[name]
	if !exists {
		return nil
	}
	job := *jd
	return &job
}

func GetJobs() []JobDefinition {
	jobs := make([]JobDefinition, 0, len(registeredJobs))
	for _, j := range registeredJobs {
		jobs = append(jobs, *j)
	}
	return jobs
}
