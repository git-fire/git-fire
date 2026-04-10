package plugins

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/git-fire/git-fire/internal/safety"
)

// CommandPlugin executes external commands
type CommandPlugin struct {
	name    string
	command string
	args    []string
	env     map[string]string
	timeout time.Duration
	when    Trigger
	failRun bool
}

// NewCommandPlugin creates a new command plugin
func NewCommandPlugin(name, command string, args []string) *CommandPlugin {
	return &CommandPlugin{
		name:    name,
		command: command,
		args:    args,
		env:     make(map[string]string),
		timeout: 5 * time.Minute, // Default 5 min timeout
		when:    TriggerOnSuccess,
	}
}

// Name returns the plugin name
func (p *CommandPlugin) Name() string {
	return p.name
}

// Type returns the plugin type
func (p *CommandPlugin) Type() PluginType {
	return PluginTypeCommand
}

// ResolveConflict implements ConflictResolver for command plugins configured
// with trigger on-merge-conflict.
func (p *CommandPlugin) ResolveConflict(ctx ConflictContext) (ConflictResolutionResult, error) {
	return p.ExecuteConflictResolution(ctx)
}

// SetTimeout sets the command timeout
func (p *CommandPlugin) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}

// SetEnv sets environment variables
func (p *CommandPlugin) SetEnv(key, value string) {
	p.env[key] = value
}

// SetTrigger sets when the plugin should run
func (p *CommandPlugin) SetTrigger(trigger Trigger) {
	p.when = trigger
}

// When returns when this plugin runs (post-run hooks vs merge-conflict, etc.).
func (p *CommandPlugin) When() Trigger {
	return p.when
}

// SetFailRun sets whether plugin failures should fail the overall run.
func (p *CommandPlugin) SetFailRun(v bool) {
	p.failRun = v
}

// FailRun returns whether plugin failures should fail the overall run.
func (p *CommandPlugin) FailRun() bool {
	return p.failRun
}

// Validate checks if the plugin is valid
func (p *CommandPlugin) Validate() error {
	if p.name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if p.command == "" {
		return fmt.Errorf("command is required")
	}

	// Check if command exists
	if _, err := exec.LookPath(p.command); err != nil {
		return fmt.Errorf("command %s not found in PATH: %w", p.command, err)
	}

	return nil
}

// ExecuteConflictResolution runs the command for merge-conflict handling and
// interprets the first line of stdout as whether the divergence was resolved.
func (p *CommandPlugin) ExecuteConflictResolution(ctx ConflictContext) (ConflictResolutionResult, error) {
	if ctx.DryRun {
		dryLine := fmt.Sprintf("[DRY RUN] Would run merge-conflict resolver: %s %s",
			p.command, strings.Join(p.args, " "))
		ctx.Logger.Info(safety.SanitizeText(dryLine))
		return ConflictResolutionResult{Resolved: false}, nil
	}

	expandedArgs := make([]string, len(p.args))
	for i, arg := range p.args {
		expandedArgs[i] = p.expandConflictVars(arg, ctx)
	}

	ctx.Logger.Info(safety.SanitizeText(fmt.Sprintf("Merge-conflict resolver: %s %s",
		p.command, strings.Join(expandedArgs, " "))))

	cmd := exec.Command(p.command, expandedArgs...)
	cmd.Dir = ctx.RepoPath
	cmd.Env = os.Environ()
	for key, value := range p.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, p.expandConflictVars(value, ctx)))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return ConflictResolutionResult{}, fmt.Errorf("failed to start command: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(p.timeout):
		_ = cmd.Process.Kill()
		return ConflictResolutionResult{}, fmt.Errorf("command timed out after %v", p.timeout)

	case err := <-done:
		if err != nil {
			sanitizedStderr := safety.SanitizeText(stderr.String())
			ctx.Logger.Error(fmt.Sprintf("Merge-conflict resolver failed: %s", sanitizedStderr), err)
			return ConflictResolutionResult{}, fmt.Errorf("command failed: %w\nStderr: %s", err, sanitizedStderr)
		}

		out := stdout.String()
		if len(strings.TrimSpace(out)) > 0 {
			ctx.Logger.Debug(fmt.Sprintf("Output: %s", safety.SanitizeText(out)))
		}

		resolved := ParseConflictResolvedLine(out)
		if resolved {
			ctx.Logger.Success("Merge-conflict resolver reported resolved")
		} else {
			ctx.Logger.Info("Merge-conflict resolver reported not resolved")
		}
		return ConflictResolutionResult{Resolved: resolved}, nil
	}
}

// Execute runs the command
func (p *CommandPlugin) Execute(ctx Context) error {
	if ctx.DryRun {
		dryLine := fmt.Sprintf("[DRY RUN] Would execute: %s %s",
			p.command, strings.Join(p.args, " "))
		ctx.Logger.Info(safety.SanitizeText(dryLine))
		return nil
	}

	// Expand variables in args
	expandedArgs := make([]string, len(p.args))
	for i, arg := range p.args {
		expandedArgs[i] = p.expandVars(arg, ctx)
	}

	ctx.Logger.Info(safety.SanitizeText(fmt.Sprintf("Executing: %s %s",
		p.command, strings.Join(expandedArgs, " "))))

	// Create command
	cmd := exec.Command(p.command, expandedArgs...)

	// Set environment
	cmd.Env = os.Environ()
	for key, value := range p.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, p.expandVars(value, ctx)))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(p.timeout):
		_ = cmd.Process.Kill()
		return fmt.Errorf("command timed out after %v", p.timeout)

	case err := <-done:
		if err != nil {
			sanitizedStderr := safety.SanitizeText(stderr.String())
			ctx.Logger.Error(fmt.Sprintf("Command failed: %s", sanitizedStderr), err)
			return fmt.Errorf("command failed: %w\nStderr: %s", err, sanitizedStderr)
		}

		if stdout.Len() > 0 {
			ctx.Logger.Debug(fmt.Sprintf("Output: %s", safety.SanitizeText(stdout.String())))
		}

		ctx.Logger.Success("Command completed successfully")
		return nil
	}
}

// Cleanup performs any cleanup
func (p *CommandPlugin) Cleanup() error {
	// Command plugins don't need cleanup
	return nil
}

// expandVars replaces template variables in a string
func (p *CommandPlugin) expandVars(s string, ctx Context) string {
	replacements := map[string]string{
		"{repo_path}":  ctx.RepoPath,
		"{repo_name}":  ctx.RepoName,
		"{branch}":     ctx.Branch,
		"{commit_sha}": ctx.CommitSHA,
		"{timestamp}":  ctx.Timestamp.Format("20060102-150405"),
		"{date}":       ctx.Timestamp.Format("2006-01-02"),
		"{time}":       ctx.Timestamp.Format("15:04:05"),
	}

	result := s
	for key, value := range replacements {
		if value == "" {
			continue
		}
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

func (p *CommandPlugin) expandConflictVars(s string, ctx ConflictContext) string {
	replacements := map[string]string{
		"{repo_path}":   ctx.RepoPath,
		"{repo_name}":   ctx.RepoName,
		"{branch}":      ctx.Branch,
		"{remote}":      ctx.Remote,
		"{local_sha}":   ctx.LocalSHA,
		"{remote_sha}":  ctx.RemoteSHA,
		"{timestamp}":   ctx.Timestamp.Format("20060102-150405"),
		"{date}":        ctx.Timestamp.Format("2006-01-02"),
		"{time}":        ctx.Timestamp.Format("15:04:05"),
	}

	result := s
	for key, value := range replacements {
		if value == "" {
			continue
		}
		result = strings.ReplaceAll(result, key, value)
	}
	return result
}
