package cmd

type ConfigKind struct {
	Kind string
}

type Pod struct {
	Name   string
	Labels struct {
		App string
		Env string
	}
	Containers struct {
		Name      string
		Image     string
		Ports     []int32
		Resources struct {
			Cpu    int32
			Memory string
		}
		Commands     []string
		VolumnMounts struct {
			Name      string
			MountPath string
		}
	}
	Volumes []string
}
