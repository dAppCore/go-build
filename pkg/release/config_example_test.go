package release

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	yaml "gopkg.in/yaml.v3"
)

// --- v0.9.0 generated usage examples ---
func ExampleConfig_PublishersIter() {
	subject := &Config{}
	_ = subject.PublishersIter()
	core.Println("Config_PublishersIter")
	// Output: Config_PublishersIter
}

func ExampleLoadConfig() {
	_, _ = LoadConfig(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("LoadConfig")
	// Output: LoadConfig
}

func ExampleLoadConfigWithMedium() {
	_, _ = LoadConfigWithMedium(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("LoadConfigWithMedium")
	// Output: LoadConfigWithMedium
}

func ExampleLoadConfigAtPath() {
	_, _ = LoadConfigAtPath(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("LoadConfigAtPath")
	// Output: LoadConfigAtPath
}

func ExampleDefaultConfig() {
	_ = DefaultConfig()
	core.Println("DefaultConfig")
	// Output: DefaultConfig
}

func ExampleTargetConfig_MarshalYAML() {
	subject := TargetConfig{OS: "linux", Arch: "amd64"}
	value, _ := subject.MarshalYAML()
	core.Println(value.(map[string]string)["arch"])
	// Output: amd64
}

func ExampleTargetConfig_UnmarshalYAML() {
	node := &yaml.Node{}
	_ = node.Encode(map[string]string{releaseTargetOSField: "linux", "arch": "amd64"})
	var subject TargetConfig
	_ = subject.UnmarshalYAML(node)
	core.Println(subject.OS)
	// Output: linux
}

func ExampleScaffoldConfig() {
	_ = ScaffoldConfig()
	core.Println("ScaffoldConfig")
	// Output: ScaffoldConfig
}

func ExampleConfig_ExpandEnv() {
	subject := &Config{}
	subject.ExpandEnv()
	core.Println("Config_ExpandEnv")
	// Output: Config_ExpandEnv
}

func ExampleConfig_SetProjectDir() {
	subject := &Config{}
	subject.SetProjectDir(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Config_SetProjectDir")
	// Output: Config_SetProjectDir
}

func ExampleConfig_SetVersion() {
	subject := &Config{}
	subject.SetVersion("v1.2.3")
	core.Println("Config_SetVersion")
	// Output: Config_SetVersion
}

func ExampleConfig_SetOutput() {
	subject := &Config{}
	subject.SetOutput(coreio.NewMemoryMedium(), core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Config_SetOutput")
	// Output: Config_SetOutput
}

func ExampleConfig_SetOutputMedium() {
	subject := &Config{}
	subject.SetOutputMedium(coreio.NewMemoryMedium())
	core.Println("Config_SetOutputMedium")
	// Output: Config_SetOutputMedium
}

func ExampleConfig_SetOutputDir() {
	subject := &Config{}
	subject.SetOutputDir(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("Config_SetOutputDir")
	// Output: Config_SetOutputDir
}

func ExampleConfigPath() {
	_ = ConfigPath(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("ConfigPath")
	// Output: ConfigPath
}

func ExampleConfigExists() {
	_ = ConfigExists(core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("ConfigExists")
	// Output: ConfigExists
}

func ExampleConfig_GetRepository() {
	subject := &Config{}
	_ = subject.GetRepository()
	core.Println("Config_GetRepository")
	// Output: Config_GetRepository
}

func ExampleConfig_GetProjectName() {
	subject := &Config{}
	_ = subject.GetProjectName()
	core.Println("Config_GetProjectName")
	// Output: Config_GetProjectName
}

func ExampleWriteConfig() {
	_ = WriteConfig(&Config{}, core.Path(core.TempDir(), "go-build-compliance"))
	core.Println("WriteConfig")
	// Output: WriteConfig
}
