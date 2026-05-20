// Package profiles supplies preset agent.Profile constructors.
//
// A profile picks two tool-name lists — ActiveTools (eager) and DeferredTools
// (lazy via TOOL_SEARCH) — an LLM target, and a system prompt. Each profile
// builds its own system prompt internally via the sysprompt package; callers
// never pass a sysprompt string in. The invariant: a distinct system prompt
// always lives behind a distinct profile constructor — never as an ad-hoc
// input — so two agents on the same Profile behave identically.
//
// Adding a new profile = one function composing name lists from the family
// Names() helpers plus a buildSysPrompt call.
package agent

import (
	"fmt"
	"slices"

	config "github.com/johnny1110/evva/configs"
	"github.com/johnny1110/evva/internal/agent/sysprompt"
	"github.com/johnny1110/evva/internal/constant"
	"github.com/johnny1110/evva/internal/llm"
	"github.com/johnny1110/evva/internal/memdir"
	"github.com/johnny1110/evva/internal/tools"
	"github.com/johnny1110/evva/internal/tools/cron"
	"github.com/johnny1110/evva/internal/tools/dev"
	"github.com/johnny1110/evva/internal/tools/fs"
	"github.com/johnny1110/evva/internal/tools/meta"
	"github.com/johnny1110/evva/internal/tools/mode"
	"github.com/johnny1110/evva/internal/tools/monitor"
	"github.com/johnny1110/evva/internal/tools/notebook"
	"github.com/johnny1110/evva/internal/tools/shell"
	"github.com/johnny1110/evva/internal/tools/skill"
	"github.com/johnny1110/evva/internal/tools/todo"
	"github.com/johnny1110/evva/internal/tools/util"
	"github.com/johnny1110/evva/internal/tools/ux"
	"github.com/johnny1110/evva/internal/tools/web"
	"github.com/johnny1110/evva/internal/toolset"
)

// AgentType enumerates the kinds of agent we know how to bootstrap.
// Profiles in agent/profiles are keyed off these values; the value also
// appears in logs to identify which kind of agent emitted a record.
type AgentType int

const (
	MAIN AgentType = iota
	EXPLORE
	GENERAL_PURPOSE
)

// String returns a short human label suitable for logs and the system prompt.
func (t AgentType) String() string {
	switch t {
	case MAIN:
		return "main"
	case EXPLORE:
		return "explore"
	case GENERAL_PURPOSE:
		return "general"
	default:
		return "unknown"
	}
}

// Profile is the configuration an Agent runs under: which kind of agent it
// is, what system prompt it presents, and which tool *names* are exposed to
// the model.
//
// Tool policy is split into two lists — this split is purely an agent-level
// scheduling decision; the tool packages themselves know nothing about it:
//
//   - ActiveTools are constructed at agent.New() and exposed to the LLM in
//     every Complete call. The model can invoke them with no preamble.
//
//   - DeferredTools are advertised to the model by name only. They are
//     materialized on demand via agent.LoadDeferred (driven by TOOL_SEARCH).
//     Listing a name here is the agent's allowlist for what may be lazily
//     loaded; a profile that omits a name forbids it entirely.
//
// Two agents with the same Profile behave identically — the loop, dispatch,
// and lifecycle are shared in the Agent type; only configuration varies.
type Profile struct {
	Type         AgentType
	SystemPrompt string

	// Tool policy
	ActiveTools   []tools.ToolName
	DeferredTools []tools.ToolName

	// LLM core
	LLMProvider constant.LLMProvider
	LLMModel    constant.Model
	LLMOptions  []llm.Option

	// Stream selects the streaming completion path. When true the agent
	// calls llm.Client.Stream and forwards each delta to the event sink
	// as KindTextChunk / KindThinkingChunk; when false it calls Complete
	// and emits a single KindText / KindThinking after the turn assembles.
	Stream bool
}

// Main returns the full-kit profile: fs/shell/meta/skill are active; the rest
// are deferred (loaded on demand via TOOL_SEARCH).
//
// The system prompt is built via sysprompt.MainAgent. Skills are advertised
// (Main is the only agent that surfaces them), and the EVVA.md / USER_PROFILE.md
// memory snapshot is threaded into the prompt under labeled headings. Callers
// pass an empty memdir.Snapshot{} when memory injection is not desired.
//
// Streaming is on by default — the user-facing UX win is large and the
// chunk adapter falls back cleanly for providers without native streaming.
// Callers who want the old buffered behavior can pass WithStream(false) at
// agent construction.
func Main(cfg *config.AppConfig, provider constant.LLMProvider, model constant.Model, skills []sysprompt.SkillRef, mem memdir.Snapshot, options []llm.Option) Profile {
	activeTools := slices.Concat(fs.Names(), shell.Names(), meta.Names(), skill.Names(), todo.Names())
	// dev env tools for collect agent feedback
	if cfg.IsDevelopment() {
		activeTools = append(activeTools, dev.Names()...)
	}
	deferredTools := slices.Concat(
		monitor.Names(),
		mode.Names(),
		notebook.Names(),
		ux.Names(),
		cron.Names(),
		web.Names(),
		util.Names(),
	)

	ctx := sysprompt.DetectContext(cfg.AppName, cfg.EvvaHome, cfg.AppEnv)
	ctx.Skills = skills
	ctx.ProjectMemory = mem.ProjectMemory
	ctx.UserProfile = mem.UserProfile
	ctx.DeferredTools = deferredToolSpecs(deferredTools)
	sp := sysprompt.MainAgent.BuildSystemPrompt(ctx)
	options = append(options, llm.WithSystem(sp))

	return Profile{
		Type:          MAIN,
		SystemPrompt:  sp,
		ActiveTools:   activeTools,
		DeferredTools: deferredTools,
		LLMProvider:   provider,
		LLMModel:      model,
		LLMOptions:    options,
		Stream:        false,
	}
}

// ResolveMainProfile is the single entry point for picking a main-tier
// Profile by persona name. Used by both bootstrap (cmd/evva/main.go) and
// the runtime /profile switch (Agent.SwitchProfile).
//
// Built-in "evva" routes through Main(...) verbatim — the same full-kit
// active/deferred tool lists, the same memdir + skills wiring.
// Disk-loaded main personas route through mainProfileFromDiskAgent which
// uses the def's own tool lists and BuildSystemPrompt body, gated by the
// def's OmitMemory / AdvertiseSkills flags from meta.yml.
//
// Empty name defaults to "evva". Unknown or non-main names return an
// error so callers (bootstrap fallback, the /profile picker) can surface
// the failure.
func ResolveMainProfile(cfg *config.AppConfig, reg *AgentRegistry, name string, skills []sysprompt.SkillRef, mem memdir.Snapshot, options []llm.Option) (Profile, error) {
	if name == "" {
		name = "evva"
	}
	if reg == nil {
		// No registry — only the built-in evva is reachable. Accept the
		// "evva" name; everything else is unknown.
		if name != "evva" {
			return Profile{}, fmt.Errorf("agent: unknown main profile %q (no registry)", name)
		}
		return Main(cfg, cfg.DefaultProvider, cfg.DefaultModel, skills, mem, options), nil
	}
	def, ok := reg.Get(name)
	if !ok {
		return Profile{}, fmt.Errorf("agent: unknown main profile %q", name)
	}
	if !def.IsMain() {
		return Profile{}, fmt.Errorf("agent: %q is not a main-tier persona", name)
	}
	if def.Name == "evva" {
		return Main(cfg, cfg.DefaultProvider, cfg.DefaultModel, skills, mem, options), nil
	}
	return mainProfileFromDiskAgent(def, cfg, cfg.DefaultProvider, cfg.DefaultModel, skills, mem, options), nil
}

// mainProfileFromDiskAgent builds a MAIN-tier Profile from a disk-loaded
// AgentDefinition. Mirrors the subagent-tier profileFromDiskAgent in
// spawn.go; the deltas are Type=MAIN, opt-in memory injection, opt-in
// skills advertisement.
//
// Tool lists come straight from the def's ActiveTools / DeferredTools
// (loaded from tools.yml). The deferred catalog is rendered into the
// prompt so disk personas see their lazy-loadable tools the same way
// built-in evva does.
func mainProfileFromDiskAgent(def sysprompt.AgentDefinition, cfg *config.AppConfig, provider constant.LLMProvider, model constant.Model, skills []sysprompt.SkillRef, mem memdir.Snapshot, options []llm.Option) Profile {
	ctx := sysprompt.DetectContext(cfg.AppName, cfg.EvvaHome, cfg.AppEnv)
	if def.AdvertiseSkills {
		ctx.Skills = skills
	}
	if !def.OmitMemory {
		ctx.ProjectMemory = mem.ProjectMemory
		ctx.UserProfile = mem.UserProfile
	}
	ctx.DeferredTools = deferredToolSpecs(def.DeferredTools)
	body := def.BuildSystemPrompt(ctx)
	sp := sysprompt.ComposeDiskMainPrompt(body, ctx, def)
	options = append(options, llm.WithSystem(sp))
	return Profile{
		Type:          MAIN,
		SystemPrompt:  sp,
		ActiveTools:   def.ActiveTools,
		DeferredTools: def.DeferredTools,
		LLMProvider:   provider,
		LLMModel:      model,
		LLMOptions:    options,
	}
}

// deferredToolSpecs flattens a list of deferred tool names into the prompt
// shape sysprompt.PromptContext consumes. Each name is resolved through
// toolset.Describe — names that don't resolve (unknown, registration race)
// are dropped rather than erroring; the resulting prompt simply omits them.
func deferredToolSpecs(names []tools.ToolName) []sysprompt.DeferredToolSpec {
	out := make([]sysprompt.DeferredToolSpec, 0, len(names))
	for _, n := range names {
		d, err := toolset.Describe(n)
		if err != nil {
			continue
		}
		out = append(out, sysprompt.DeferredToolSpec{
			Name:        d.Name,
			Description: d.Description,
			Schema:      d.Schema,
		})
	}
	return out
}

// Explore returns a read-only profile: just READ_FILE / GREP / TREE, plus
// WEB_SEARCH for docs lookup. Useful for sub-agents whose job is to inspect
// without risk of modification.
//
// The Explore prompt is self-contained (mirrors ref TS Explore agent) and
// does not include EVVA.md / USER_PROFILE.md — sysprompt.ExploreAgent
// declares OmitMemory: true.
func Explore(cfg *config.AppConfig, provider constant.LLMProvider, model constant.Model, options []llm.Option) Profile {
	ctx := sysprompt.DetectContext(cfg.AppName, cfg.EvvaHome, cfg.AppEnv)
	sp := sysprompt.ExploreAgent.BuildSystemPrompt(ctx)
	options = append(options, llm.WithSystem(sp))

	return Profile{
		Type:         EXPLORE,
		SystemPrompt: sp,
		ActiveTools:  []tools.ToolName{tools.READ_FILE, tools.WEB_SEARCH, tools.GLOB, tools.TREE, tools.GREP, tools.JSON_QUERY},
		LLMProvider:  provider,
		LLMModel:     model,
		LLMOptions:   options,
	}
}

// General returns a minimal profile carrying only the tool names the caller
// supplies as active. No deferred tools. Useful for narrow-purpose sub-agents.
//
// Like Explore, the General prompt does not include EVVA.md / USER_PROFILE.md.
func General(cfg *config.AppConfig, provider constant.LLMProvider, model constant.Model, options []llm.Option, toolset ...tools.ToolName) Profile {
	ctx := sysprompt.DetectContext(cfg.AppName, cfg.EvvaHome, cfg.AppEnv)
	sp := sysprompt.GeneralAgent.BuildSystemPrompt(ctx)
	options = append(options, llm.WithSystem(sp))

	return Profile{
		Type:         GENERAL_PURPOSE,
		SystemPrompt: sp,
		ActiveTools:  toolset,
		LLMProvider:  provider,
		LLMModel:     model,
		LLMOptions:   options,
	}
}
