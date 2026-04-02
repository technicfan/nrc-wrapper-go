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
	GetCurrentInstanceDetails() (Minecraft, error)
	GetInstances([]string, []string, string) ([]Instance, error)
	Exists() bool
	IsRunning() bool
	ReplaceNormal(string)
	NormalDir() string
}

type launcher_data struct {
	name string
	path string
	instance_dir string
	flatpak_id string
	replaced bool
	normal_dir string
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

func (data *launcher_data) ReplaceNormal(normal string) {
	if data.flatpak_id != "" {
		data.replaced = true
		data.normal_dir = normal
	}
}

func (data *launcher_data) NormalDir() string {
	return data.normal_dir
}
