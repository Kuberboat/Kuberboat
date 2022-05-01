package apiserver

// PodLegacy is the information of a deleted pod that might be used by some controllers for event handling.
type PodLegacy struct {
	// DeploymentName is the name of the deployment managing the deleted pod.
	// Empty if the pod wasn't managed by any deployment.
	DeploymentName string
}

type LegacyManager interface {
	// GetPodLegacyByName gets the legacy of a deleted pod indexed by name.
	GetPodLegacyByName(name string) *PodLegacy
	// SetPodLegacy sets a pod's legacy. The legacy will contain as much information as the ComponentManager can offer,
	// so the caller doesn't need to worry about how to populate the legacy.
	SetPodLegacy(name string)
	// DeletePodLegacy removes a pod's legacy.
	DeletePodLegacyByName(name string)
}

type legacyManagerInner struct {
	componentManager ComponentManager
	podLegacy        map[string]*PodLegacy
}

func NewLegacyManager(componentManager ComponentManager) LegacyManager {
	return &legacyManagerInner{
		componentManager: componentManager,
		podLegacy:        map[string]*PodLegacy{},
	}
}

func (m *legacyManagerInner) GetPodLegacyByName(name string) *PodLegacy {
	return m.podLegacy[name]
}

func (m *legacyManagerInner) SetPodLegacy(name string) {
	legacy := &PodLegacy{}
	if deployment := m.componentManager.GetDeploymentByPodName(name); deployment != nil {
		legacy.DeploymentName = deployment.Name
	}
	m.podLegacy[name] = legacy
}

func (m *legacyManagerInner) DeletePodLegacyByName(name string) {
	delete(m.podLegacy, name)
}
