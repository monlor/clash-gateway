package netns

type Plan struct {
	ContainerID string
	Actions     []string
}

func BuildAttachPlan(containerID, gatewayName, managedNetwork string) Plan {
	return Plan{
		ContainerID: containerID,
		Actions: []string{
			"connect container to managed bridge network " + managedNetwork,
			"prepare transparent TCP + DNS routing for gateway " + gatewayName,
		},
	}
}
