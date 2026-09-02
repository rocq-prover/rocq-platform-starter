package installer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/justme0606/rocq-platform-starter/linux/internal/vscode"
	"github.com/justme0606/rocq-platform-starter/linux/internal/workspace"
)

// DockerConfig holds all parameters for the Docker installation pipeline.
type DockerConfig struct {
	Image         string // full image reference, e.g. "ghcr.io/rocq-prover/rocq-platform_ide:2026.01"
	VsrocqtopPath string // path to vsrocqtop inside the container
	User          string // container user, e.g. "rocq"
	Templates     fs.FS
	OnStep        StepFunc
	Logger        *Logger
}

// DockerResult holds information about the Docker installation outcome.
type DockerResult struct {
	VSCodeFound bool
}

// RunDocker executes the Docker installation pipeline.
// Steps:
//  1. Check Docker is available
//  2. Pull the Docker image
//  3. Create workspace + .devcontainer/devcontainer.json
//  4. Configure VSCode (install Dev Containers + vsrocq extensions)
//  5. Open VSCode in the workspace
func RunDocker(cfg *DockerConfig) (*DockerResult, error) {
	result := &DockerResult{}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	workspaceDir := filepath.Join(home, WorkspaceName)

	// Step 1: Check Docker
	cfg.OnStep(1, "Checking for Docker...", 0.0)
	if err := ensureDocker(cfg.Logger); err != nil {
		return nil, fmt.Errorf("docker: %w", err)
	}
	cfg.OnStep(1, "Docker found.", 1.0)

	// Step 2: Pull image
	cfg.OnStep(2, fmt.Sprintf("Pulling %s...", cfg.Image), 0.0)
	if err := pullImage(cfg.Image, cfg.Logger); err != nil {
		return nil, fmt.Errorf("docker pull: %w", err)
	}
	cfg.OnStep(2, "Image pulled.", 1.0)

	// Step 3: Create workspace + devcontainer.json
	cfg.OnStep(3, "Creating workspace...", 0.0)
	cfg.Logger.Log("Creating workspace at %s", workspaceDir)
	if err := workspace.Create(workspaceDir, cfg.Templates); err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	if err := writeDevcontainer(workspaceDir, cfg.Image, cfg.VsrocqtopPath, cfg.User); err != nil {
		return nil, fmt.Errorf("devcontainer: %w", err)
	}
	cfg.Logger.Log("Workspace and devcontainer.json created")
	cfg.OnStep(3, "Workspace created.", 1.0)

	// Step 4: Configure VSCode
	cfg.OnStep(4, "Configuring VSCode...", 0.0)
	codeBin, err := vscode.FindCode()
	if err != nil {
		cfg.Logger.Log("VSCode not found: %v", err)
		cfg.OnStep(4, "VSCode not found.", 1.0)
		result.VSCodeFound = false
		return result, nil
	}
	result.VSCodeFound = true

	cfg.Logger.Log("VSCode CLI: %s", codeBin)

	// Install Dev Containers extension (vsrocq is installed inside the container
	// by devcontainer.json, not on the host — installing it on the host causes
	// errors when it tries to find a local vsrocqtop that doesn't exist)
	if err := vscode.InstallExtension(codeBin, "ms-vscode-remote.remote-containers"); err != nil {
		cfg.Logger.Log("WARNING: Dev Containers extension install failed: %v", err)
	}
	cfg.OnStep(4, "VSCode configured.", 1.0)

	// Step 5: Open VSCode (disable vsrocq on the host — it will be installed
	// inside the container by devcontainer.json)
	cfg.OnStep(5, "Opening VSCode...", 0.0)
	cfg.Logger.Log("Opening VSCode with workspace %s", workspaceDir)
	if err := vscode.OpenWorkspaceWithDisabledExtensions(codeBin, workspaceDir, []string{
		"rocq-prover.vsrocq",
		"coq-community.vscoq",
	}); err != nil {
		cfg.Logger.Log("WARNING: failed to open VSCode: %v", err)
	}
	cfg.OnStep(5, "Done!", 1.0)

	return result, nil
}

// ensureDocker checks that docker is in PATH and the daemon is running.
func ensureDocker(logger *Logger) error {
	path, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker not found in PATH. Please install Docker: https://docs.docker.com/get-docker/")
	}
	logger.Log("docker found: %s", path)

	// Check that the daemon is running
	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		logger.Log("docker info failed: %s", string(out))
		return fmt.Errorf("Docker daemon is not running. Please start Docker and try again.")
	}
	logger.Log("Docker daemon is running")
	return nil
}

// pullImage runs docker pull and logs the output.
func pullImage(image string, logger *Logger) error {
	logger.Log("Pulling image: %s", image)

	cmd := exec.Command("docker", "pull", image)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start docker pull: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		logger.Log("[docker] %s", scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("docker pull failed: %w", err)
	}

	logger.Log("Image pulled successfully")
	return nil
}

// devcontainerJSON is the structure for .devcontainer/devcontainer.json.
type devcontainerJSON struct {
	Name                string                 `json:"name"`
	Image               string                 `json:"image"`
	Customizations      map[string]interface{} `json:"customizations"`
	RemoteUser          string                 `json:"remoteUser"`
	UpdateRemoteUserUID bool                   `json:"updateRemoteUserUID"`
	WorkspaceFolder     string                 `json:"workspaceFolder"`
	WorkspaceMount      string                 `json:"workspaceMount"`
	PostCreateCmd       string                 `json:"postCreateCommand"`
}

// writeDevcontainer creates .devcontainer/devcontainer.json in the workspace.
func writeDevcontainer(workspaceDir, image, vsrocqtopPath, user string) error {
	devcontainerDir := filepath.Join(workspaceDir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0o755); err != nil {
		return fmt.Errorf("create .devcontainer dir: %w", err)
	}

	dc := devcontainerJSON{
		Name:  "Rocq Platform",
		Image: image,
		Customizations: map[string]interface{}{
			"vscode": map[string]interface{}{
				"extensions": []string{
					"rocq-prover.vsrocq",
				},
				"settings": map[string]string{
					"vsrocq.path": vsrocqtopPath,
				},
			},
		},
		RemoteUser:      user,
		WorkspaceFolder: fmt.Sprintf("/home/%s/workspace", user),
		WorkspaceMount:  fmt.Sprintf("source=${localWorkspaceFolder},target=/home/%s/workspace,type=bind,consistency=cached", user),
		PostCreateCmd:   "rocq --version",
	}

	data, err := json.MarshalIndent(dc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}
	data = append(data, '\n')

	dest := filepath.Join(devcontainerDir, "devcontainer.json")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("write devcontainer.json: %w", err)
	}

	return nil
}
