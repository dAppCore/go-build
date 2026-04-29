package service

import (
	"net/http"
	"sort"

	coreapi "dappco.re/go/api"
	providerpkg "dappco.re/go/api/pkg/provider"
	"dappco.re/go/build/pkg/build"
	"dappco.re/go/ws"
	"github.com/gin-gonic/gin"
)

type toolDescriber interface {
	Tools() []coreapi.ToolDescriptor
}

func defaultNewMCPServer(cfg Config, registry *providerpkg.Registry, hub *ws.Hub) coreapi.RouteGroup {
	if registry == nil {
		registry = providerpkg.NewRegistry()
	}

	bridge := coreapi.NewToolBridge("/api/v1/mcp")

	bridge.Add(coreapi.ToolDescriptor{
		Name:        "providers_list",
		Description: "List daemon providers, channels, and GUI elements exposed to MCP clients.",
		Group:       "mcp",
	}, func(c *gin.Context) {
		c.JSON(http.StatusOK, coreapi.OK(map[string]any{
			"providers": registry.Info(),
		}))
	})

	bridge.Add(coreapi.ToolDescriptor{
		Name:        "daemon_status",
		Description: "Return daemon addresses, watch settings, provider inventory, and MCP tool metadata.",
		Group:       "mcp",
	}, func(c *gin.Context) {
		c.JSON(http.StatusOK, coreapi.OK(map[string]any{
			"name":              cfg.Name,
			"project_dir":       cfg.ProjectDir,
			"api_addr":          cfg.APIAddr,
			"health_addr":       cfg.HealthAddr,
			"auto_rebuild":      cfg.AutoRebuild,
			"watch_paths":       append([]string(nil), cfg.WatchPaths...),
			"watch_interval":    cfg.WatchInterval.String(),
			"schedule_interval": cfg.ScheduleInterval.String(),
			"providers":         registry.Info(),
			"tools":             mcpToolNames(bridge),
		}))
	})

	bridge.Add(coreapi.ToolDescriptor{
		Name:        "project_discover",
		Description: "Inspect the current project and return the build discovery summary used by the daemon.",
		Group:       "build",
	}, func(c *gin.Context) {
		discovery := discoverProject(cfg.ProjectDir)
		if !discovery.OK {
			c.JSON(http.StatusInternalServerError, coreapi.FailWithDetails(
				"discover_failed",
				"Failed to inspect the daemon project",
				map[string]any{"error": discovery.Error()},
			))
			return
		}
		result := discovery.Value.(*build.DiscoveryResult)

		c.JSON(http.StatusOK, coreapi.OK(map[string]any{
			"project_dir":       cfg.ProjectDir,
			"types":             discoveryTypes(result),
			"primary_stack":     result.PrimaryStack,
			"suggested_stack":   result.SuggestedStack,
			"configured_type":   result.ConfiguredType,
			"has_frontend":      result.HasFrontend,
			"has_docs_config":   result.HasDocsConfig,
			"has_deno_manifest": result.HasDenoManifest,
			"distro":            result.Distro,
		}))
	})

	bridge.Add(coreapi.ToolDescriptor{
		Name:        "build_run",
		Description: "Trigger the daemon build pipeline immediately using the watched-build path.",
		Group:       "build",
	}, func(c *gin.Context) {
		sendEvent(hub, "build.started", map[string]any{
			"projectDir": cfg.ProjectDir,
			"source":     "mcp.build_run",
		})

		built := runWatchedBuild(c.Request.Context(), cfg.ProjectDir)
		if !built.OK {
			sendEvent(hub, "build.failed", map[string]any{
				"projectDir": cfg.ProjectDir,
				"source":     "mcp.build_run",
				"error":      built.Error(),
			})
			c.JSON(http.StatusInternalServerError, coreapi.FailWithDetails(
				"build_failed",
				"Daemon build failed",
				map[string]any{"error": built.Error()},
			))
			return
		}

		sendEvent(hub, "build.complete", map[string]any{
			"projectDir": cfg.ProjectDir,
			"source":     "mcp.build_run",
		})
		c.JSON(http.StatusOK, coreapi.OK(map[string]any{
			"project_dir": cfg.ProjectDir,
			"status":      "complete",
		}))
	})

	return bridge
}

func mcpToolNames(group coreapi.RouteGroup) []string {
	describer, ok := group.(toolDescriber)
	if !ok {
		return nil
	}

	descriptors := describer.Tools()
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Name == "" {
			continue
		}
		names = append(names, descriptor.Name)
	}
	sort.Strings(names)
	return names
}
