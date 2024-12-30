package source

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jorenkoyen/conter/manager/types"
	"github.com/jorenkoyen/go-logger"
	"github.com/jorenkoyen/go-logger/log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	DefaultBranch             = "master"
	DefaultDepth              = "1"
	DefaultDockerfileLocation = "Dockerfile"

	BuildInternalDirectory = ".conter"
	BuildScriptName        = "build.sh"
	LogOutputName          = "build.log"
	ImageOutputName        = "build.image"
)

type Builder struct {
	logger    *logger.Logger
	writer    *BashWriter
	directory string
	isError   bool
	image     string

	// configuration
	branch     string
	depth      string
	dockerfile string
}

// NewBuilder creates a new builder instance which is able to build a docker image from a repository.
func NewBuilder() *Builder {
	return &Builder{
		logger: log.WithName("builder"),
		writer: new(BashWriter),
	}
}

// prepare will set up the builder.
func (b *Builder) prepare() error {
	dir, err := os.MkdirTemp("", "*-build")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	b.logger.Debugf("Temporary directory for build has been created (dir=%s)", dir)
	b.directory = dir
	return nil
}

// cleanup will remove any lingering resources from the builder.
func (b *Builder) cleanup() {
	if b.directory != "" {
		if !b.isError {
			b.logger.Debugf("Cleaning up temporary directory (dir=%s)", b.directory)
			if err := os.RemoveAll(b.directory); err != nil {
				b.logger.Warningf("Failed to remove temporary directory (dir=%s) : %v", b.directory, err)
			}
		} else {
			b.logger.Warningf("Not cleaning up directory, as error occurred during build (dir=%s)", b.directory)
		}
	} else {
		b.logger.Tracef("No cleanup required, no directory has been created")
	}
}

// optsOrDefault will return the configured opts value or the default value if not supplied.
func (b *Builder) optsOrDefault(source types.Source, key string, defaultValue string) string {
	if source.Opts == nil {
		return defaultValue
	}

	value := source.Opts[key]
	if value == "" {
		return defaultValue
	} else {
		return value
	}
}

func (b *Builder) prepareGitClone(source types.Source) {
	args := []string{
		"clone",
		"--single-branch",
		"--branch", b.branch,
		"--depth", b.depth,
		source.URI,
		"repository",
	}

	b.writer.Command("git", args...)
	b.writer.Cd("repository")
}

func (b *Builder) prepareDockerBuild(project string, name string) {
	// use current short commit as variable
	b.writer.EnvVariableEval("CONTER_IMAGE_TAG", "git rev-parse --short HEAD")
	image := fmt.Sprintf("conter/%s/%s:%s", project, name, b.writer.EnvVariableKey("CONTER_IMAGE_TAG"))
	b.writer.EnvVariable("CONTER_IMAGE", image)

	// setup docker build command
	args := []string{
		"buildx", "build",
		"--tag", b.writer.EnvVariableKey("CONTER_IMAGE"),
		"--file", b.dockerfile,
		".", // build context
	}

	b.writer.Command("docker", args...)

	// write output of image name to file
	b.writer.PipeToFile(
		fmt.Sprintf("echo %s", b.writer.EnvVariableKey("CONTER_IMAGE")),
		filepath.Join(b.directory, BuildInternalDirectory, ImageOutputName),
	)
}

// Build will clone the repository and build the container image for the service.
func (b *Builder) Build(ctx context.Context, service types.Service) (string, error) {
	if err := b.prepare(); err != nil {
		return "", err
	}

	defer b.cleanup()

	// setup defaults
	b.dockerfile = b.optsOrDefault(service.Source, "dockerfile", DefaultDockerfileLocation)
	b.branch = b.optsOrDefault(service.Source, "branch", DefaultBranch)
	b.depth = b.optsOrDefault(service.Source, "depth", DefaultDepth)

	// create bash script:
	// -> clone repository
	b.prepareGitClone(service.Source)
	// -> start docker build
	b.prepareDockerBuild(service.Ingress.TargetProject, service.Name)

	// execute script
	if err := b.ExecuteScript(ctx); err != nil {
		return "", err
	}

	return b.image, nil
}

func (b *Builder) ExecuteScript(ctx context.Context) error {
	conterDirectory := filepath.Join(b.directory, BuildInternalDirectory)
	scriptFileLocation := filepath.Join(conterDirectory, BuildScriptName)
	logFileLocation := filepath.Join(conterDirectory, LogOutputName)
	imageFileLocation := filepath.Join(conterDirectory, ImageOutputName)

	// create conter directory
	if err := os.MkdirAll(conterDirectory, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create internal conter directory: %w", err)
	}

	// write script output to 'script.sh' file
	script := b.writer.Script(true)
	if err := os.WriteFile(scriptFileLocation, []byte(script), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create build script: %w", err)
	}

	// create log file
	logOutputFile, err := os.Create(logFileLocation)
	if err != nil {
		return fmt.Errorf("faield to create build log file: %w", err)
	}

	defer logOutputFile.Close()

	cmd := exec.CommandContext(ctx, "bash", "-c", scriptFileLocation)
	cmd.Dir = b.directory
	cmd.Stdout = logOutputFile
	cmd.Stderr = logOutputFile

	if err = cmd.Run(); err != nil {
		b.isError = true // prevents the build directory from being deleted
		return fmt.Errorf("build failed, see log output (file=%s) for more details: %w", logFileLocation, err)
	}

	content, err := os.ReadFile(imageFileLocation)
	if err != nil {
		return fmt.Errorf("failed to read build image: %w", err)
	}

	b.image = strings.Trim(string(content), "\n")
	return nil
}

type BashWriter struct {
	buffer bytes.Buffer
}

func (w *BashWriter) Command(command string, args ...string) {
	list := []string{
		command,
	}

	list = append(list, args...)
	w.Line(strings.Join(list, " "))
}

func (w *BashWriter) EnvVariableEval(name string, eval string) {
	w.Line(fmt.Sprintf("%s=$(%s)", name, eval))
}

func (w *BashWriter) EnvVariable(name string, value string) {
	w.Line(fmt.Sprintf("%s=\"%s\"", name, value))
}

func (w *BashWriter) EnvVariableKey(name string) string {
	return "$" + name
}

func (w *BashWriter) Cd(path string) {
	w.Line(fmt.Sprintf("cd \"%s\"", path))
}

func (w *BashWriter) Line(line string) {
	w.buffer.WriteString(line + "\n")
}

func (w *BashWriter) PipeToFile(command string, file string) {
	w.Line(fmt.Sprintf("%s > \"%s\"", command, file))
}

func (w *BashWriter) Script(trace bool) string {
	var buf strings.Builder

	buf.WriteString("#!/usr/bin/env bash\n\n")

	if trace {
		buf.WriteString("set -o xtrace\n")
	}

	buf.WriteString("if set -o | grep pipefail > /dev/null; then set -o pipefail; fi; set -o errexit\n")
	buf.WriteString("set +o noclobber\n")

	buf.WriteString("\n# script content\n")
	buf.WriteString(w.buffer.String())
	buf.WriteString("# end script content\n")

	buf.WriteString("exit 0\n")

	return buf.String()
}
