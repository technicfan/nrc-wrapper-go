package launchers

type Minecraft struct {
	profile       string
	version       string
	loader        string
	loader_version string
	username      string
	uuid          string
	token         string
}

func (minecraft Minecraft) Profile() string {
	return minecraft.profile
}

func (minecraft Minecraft) Version() string {
	return minecraft.version
}

func (minecraft Minecraft) Loader() string {
	return minecraft.loader
}

func (minecraft Minecraft) LoaderVersion() string {
	return minecraft.loader_version
}

func (minecraft Minecraft) Username() string {
	return minecraft.username
}

func (minecraft Minecraft) Uuid() string {
	return minecraft.uuid
}

func (minecraft Minecraft) Token() string {
	return minecraft.token
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
