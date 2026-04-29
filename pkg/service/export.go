package service

import (
	"encoding/xml"
	"strconv"

	core "dappco.re/go"
)

// Export renders a native service definition for cfg.
func Export(cfg Config, format string) (ExportedConfig, error) {
	cfg = cfg.Normalized()

	nativeFormat, err := ResolveNativeFormat(format)
	if err != nil {
		return ExportedConfig{}, err
	}

	switch nativeFormat {
	case NativeFormatSystemd:
		return ExportedConfig{
			Format:   nativeFormat,
			Filename: cfg.Name + ".service",
			Content:  renderSystemd(cfg),
		}, nil
	case NativeFormatLaunchd:
		return ExportedConfig{
			Format:   nativeFormat,
			Filename: cfg.Name + ".plist",
			Content:  renderLaunchd(cfg),
		}, nil
	case NativeFormatWindows:
		return ExportedConfig{
			Format:   nativeFormat,
			Filename: cfg.Name + ".ps1",
			Content:  renderWindows(cfg),
		}, nil
	default:
		return ExportedConfig{}, core.Errorf("unsupported native service format: %s", nativeFormat)
	}
}

func renderSystemd(cfg Config) string {
	b := core.NewBuilder()
	b.WriteString("[Unit]\n")
	b.WriteString("Description=" + cfg.Description + "\n")
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")

	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("WorkingDirectory=" + cfg.WorkingDirectory + "\n")
	b.WriteString("ExecStart=" + systemdCommand(cfg.Executable, cfg.Arguments) + "\n")
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")
	b.WriteString("Environment=CORE_BUILD_SERVICE=1\n")
	b.WriteString("Environment=CORE_BUILD_PROJECT_DIR=" + strconv.Quote(cfg.ProjectDir) + "\n")
	b.WriteString("Environment=CORE_BUILD_API_ADDR=" + strconv.Quote(cfg.APIAddr) + "\n")
	b.WriteString("Environment=CORE_BUILD_HEALTH_ADDR=" + strconv.Quote(cfg.HealthAddr) + "\n")
	b.WriteString("SyslogIdentifier=" + cfg.Name + "\n\n")

	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String()
}

func renderLaunchd(cfg Config) string {
	b := core.NewBuilder()
	b.WriteString(xml.Header)
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString("<dict>\n")
	b.WriteString("  <key>Label</key>\n")
	b.WriteString("  <string>" + xmlEscape(cfg.Name) + "</string>\n")
	b.WriteString("  <key>ProgramArguments</key>\n")
	b.WriteString("  <array>\n")
	b.WriteString("    <string>" + xmlEscape(cfg.Executable) + "</string>\n")
	for _, arg := range cfg.Arguments {
		b.WriteString("    <string>" + xmlEscape(arg) + "</string>\n")
	}
	b.WriteString("  </array>\n")
	b.WriteString("  <key>WorkingDirectory</key>\n")
	b.WriteString("  <string>" + xmlEscape(cfg.WorkingDirectory) + "</string>\n")
	b.WriteString("  <key>RunAtLoad</key>\n")
	b.WriteString("  <true/>\n")
	b.WriteString("  <key>KeepAlive</key>\n")
	b.WriteString("  <true/>\n")
	b.WriteString("  <key>EnvironmentVariables</key>\n")
	b.WriteString("  <dict>\n")
	b.WriteString("    <key>CORE_BUILD_SERVICE</key>\n")
	b.WriteString("    <string>1</string>\n")
	b.WriteString("    <key>CORE_BUILD_PROJECT_DIR</key>\n")
	b.WriteString("    <string>" + xmlEscape(cfg.ProjectDir) + "</string>\n")
	b.WriteString("    <key>CORE_BUILD_API_ADDR</key>\n")
	b.WriteString("    <string>" + xmlEscape(cfg.APIAddr) + "</string>\n")
	b.WriteString("    <key>CORE_BUILD_HEALTH_ADDR</key>\n")
	b.WriteString("    <string>" + xmlEscape(cfg.HealthAddr) + "</string>\n")
	b.WriteString("  </dict>\n")
	b.WriteString("  <key>StandardOutPath</key>\n")
	b.WriteString("  <string>" + xmlEscape(core.PathJoin(core.PathDir(cfg.PIDFile), cfg.Name+".out.log")) + "</string>\n")
	b.WriteString("  <key>StandardErrorPath</key>\n")
	b.WriteString("  <string>" + xmlEscape(core.PathJoin(core.PathDir(cfg.PIDFile), cfg.Name+".err.log")) + "</string>\n")
	b.WriteString("</dict>\n")
	b.WriteString("</plist>\n")
	return b.String()
}

func renderWindows(cfg Config) string {
	b := core.NewBuilder()
	b.WriteString("$ErrorActionPreference = \"Stop\"\n")
	b.WriteString("$serviceName = " + strconv.Quote(cfg.Name) + "\n")
	b.WriteString("$displayName = " + strconv.Quote(cfg.DisplayName) + "\n")
	b.WriteString("$description = " + strconv.Quote(cfg.Description) + "\n")
	b.WriteString("$binary = " + strconv.Quote(cfg.Executable) + "\n")
	b.WriteString("$arguments = " + strconv.Quote(core.Join(" ", cfg.Arguments...)) + "\n")
	b.WriteString("sc.exe create $serviceName binPath= ('\"' + $binary + '\" ' + $arguments) start= auto\n")
	b.WriteString("sc.exe description $serviceName $description\n")
	return b.String()
}

func systemdCommand(executable string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, strconv.Quote(executable))
	for _, arg := range args {
		parts = append(parts, strconv.Quote(arg))
	}
	return core.Join(" ", parts...)
}

func xmlEscape(value string) string {
	b := core.NewBuilder()
	if err := xml.EscapeText(b, []byte(value)); err != nil {
		return value
	}
	return b.String()
}
