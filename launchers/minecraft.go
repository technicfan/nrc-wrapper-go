package launchers

type Minecraft struct {
	Profile       string
	Version       string
	Loader        string
	LoaderVersion string
	Username      string
	Uuid          string
	Token         string
}

type Launcher interface {
	Id() string
	Name() string
	Dir() string
	InstanceDir() string
	GetDetails() (Minecraft, error)
	GetInstances([]string, []string, string) ([]Instance, error)
	Exists() bool
}

type launcher_data struct {
	name string
	path string
	instance_dir string
	flatpak_id string
}

func (data launcher_data) Name() string {
	return data.name
}

func (data launcher_data) Dir() string {
	return data.path
}

func (data launcher_data) InstanceDir() string {
	return data.instance_dir
}
