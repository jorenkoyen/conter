package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jorenkoyen/go-logger"
	"github.com/urfave/cli/v2"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

const (
	ConfigDirectory = "/etc/conter"
	DataDirectory   = "/var/lib/conter"
	ServiceName     = "conter"
	SystemdLocation = "/etc/systemd/system/conter.service"
	UnitTemplate    = `[Unit]
Description=A minimal container management system for small scale web deployments
After=network.target

[Service]
ExecStart={{.Binary}}
User=conter
Group=conter
Restart=always

[Install]
WantedBy=multi-user.target`
)

func service() *cli.Command {
	return &cli.Command{
		Name:  "service",
		Usage: "Manage the systemctl service for Conter",
		Before: func(c *cli.Context) error {
			if runtime.GOOS != "linux" {
				return errors.New("service utilities are only available for Linux")
			}
			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:  "logs",
				Usage: "Show the logs of Conter",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "follow",
						Aliases: []string{"f"},
						Usage:   "If the logs should be followed live",
					},
				},
				Action: serviceLogsHandler,
			},
			{
				Name:   "status",
				Usage:  "Show the current status of the Conter service",
				Action: statusCheckHandler,
			},
			{
				Name:   "install",
				Usage:  "Install the Conter systemctl service",
				Action: installServiceHandler,
			},
		},
	}
}

// serviceLogsHandler runs `journalctl -u` for the service and optionally follows the logs
func serviceLogsHandler(c *cli.Context) error {
	follow := c.Bool("follow")

	// Create the base command
	name := fmt.Sprintf("%s.service", ServiceName)
	args := []string{"-u", name}
	if follow {
		args = append(args, "-f") // Add the follow option
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout // Redirects the command's stdout to the program's stdout
	cmd.Stderr = os.Stderr // Redirects the command's stderr to the program's stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to show logs for service %s: %v", name, err)
	}
	return nil
}

// statusCheckHandler runs `systemctl status` for the service and outputs the content to stdout
func statusCheckHandler(c *cli.Context) error {
	name := fmt.Sprintf("%s.service", ServiceName)
	cmd := exec.Command("systemctl", "status", name)
	cmd.Stdout = os.Stdout // Redirects the command's stdout to the program's stdout
	cmd.Stderr = os.Stderr // Redirects the command's stderr to the program's stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to get status of service %s: %v", name, err)
	}
	return nil
}

func installServiceHandler(c *cli.Context) error {
	if os.Getuid() != 0 {
		return errors.New("insufficient permissions, please run as root or with sudo")
	}

	if !systemdFileExists() {
		// create users if not exists
		if err := createUsersIfNotExists(); err != nil {
			return err
		}

		location := getBinaryLocation()
		if location == "" {
			return errors.New("unable to find 'conter' binary, please check you $PATH configuration")
		}

		content := renderSystemdFile(location)
		if err := os.WriteFile(SystemdLocation, content, 0644); err != nil {
			return fmt.Errorf("failed to write systemd unit file: %w", err)
		}

		if err := reloadSystemctlDaemon(); err != nil {
			return fmt.Errorf("failed to reload systemctl daemon: %w", err)
		}

		if err := ensureDirExistsAndSetPermissions(ConfigDirectory); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		if err := ensureDirExistsAndSetPermissions(DataDirectory); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}

		if err := generateBaseConfig(); err != nil {
			return fmt.Errorf("failed to generate base config: %w", err)
		}
	}

	// enable & start system unit file
	if err := enableAndStartService(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

func getBinaryLocation() string {
	cmd := exec.Command("which", ServiceName)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// checkUserExists checks if a user already exists on the system
func checkUserExists(username string) bool {
	cmd := exec.Command("id", "-u", username)
	err := cmd.Run()
	return err == nil // If no error, the user exists
}

// checkGroupExists checks if a group already exists on the system
func checkGroupExists(groupname string) bool {
	cmd := exec.Command("getent", "group", groupname)
	err := cmd.Run()
	return err == nil // If no error, the group exists
}

// createUserAndGroup creates a Linux user and group if they do not already exist
func createUsersIfNotExists() error {
	// Check and create group if it does not exist
	if !checkGroupExists(ServiceName) {
		cmd := exec.Command("groupadd", ServiceName)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to create group %s: %v", ServiceName, err)
		}
	}

	// Check and create user if it does not exist
	if !checkUserExists(ServiceName) {
		cmd := exec.Command("useradd", "-r", "-g", ServiceName, "-s", "/bin/false", ServiceName)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to create user %s: %v", ServiceName, err)
		}
	}

	return nil
}

// reloadSystemctlDaemon reloads the systemctl daemon to apply changes to service files
func reloadSystemctlDaemon() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to reload systemctl daemon: %v", err)
	}
	return nil
}

// enableAndStartService enables and starts a systemd service
func enableAndStartService() error {
	name := fmt.Sprintf("%s.service", ServiceName)
	// Enable the service
	enableCmd := exec.Command("systemctl", "enable", name)
	if err := enableCmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service %s: %v", name, err)
	}

	// Start the service
	startCmd := exec.Command("systemctl", "start", name)
	if err := startCmd.Run(); err != nil {
		return fmt.Errorf("failed to start service %s: %v", name, err)
	}

	return nil
}

func renderSystemdFile(binary string) []byte {
	once := sync.OnceValue(func() *template.Template {
		// delayed parsing until command for install is actually invoked.
		return template.Must(template.New("conter.service").Parse(UnitTemplate))
	})

	var buf bytes.Buffer
	t := once()
	_ = t.Execute(&buf, map[string]string{
		"Binary": binary,
	})
	return buf.Bytes()
}

func systemdFileExists() bool {
	_, err := os.Stat(SystemdLocation)
	return err == nil
}

// ensureDirExistsAndSetPermissions ensures the directory exists, creates it if not, and sets the owner and group.
func ensureDirExistsAndSetPermissions(dir string) error {
	// Check if the directory exists
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// Create the directory if it does not exist
		// 0755: standard permissions for directories
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	} else if err != nil {
		return fmt.Errorf("error checking directory %s: %v", dir, err)
	}

	// Set the owner and group
	err = setOwnerAndGroup(dir, ServiceName, ServiceName)
	if err != nil {
		return err
	}

	return nil
}

// setOwnerAndGroup sets the owner and group of the directory.
func setOwnerAndGroup(path, owner, group string) error {
	// Get user by username
	userInfo, err := user.Lookup(owner)
	if err != nil {
		return fmt.Errorf("failed to lookup user %s: %v", owner, err)
	}
	// Get group by name
	groupInfo, err := user.LookupGroup(group)
	if err != nil {
		return fmt.Errorf("failed to lookup group %s: %v", group, err)
	}

	// Convert user and group IDs to integers
	uid, err := strconv.Atoi(userInfo.Uid)
	if err != nil {
		return fmt.Errorf("failed to convert user ID to integer: %v", err)
	}

	gid, err := strconv.Atoi(groupInfo.Gid)
	if err != nil {
		return fmt.Errorf("failed to convert group ID to integer: %v", err)
	}

	// Set the owner and group using os.Chown
	err = os.Chown(path, uid, gid)
	if err != nil {
		return fmt.Errorf("failed to change owner and group for %s: %v", path, err)
	}

	return nil
}

// generateBaseConfig prompts the user for their email and creates the initial required configuration.
func generateBaseConfig() error {
	reader := bufio.NewReader(os.Stdin)

	// Prompt the user for their email
	fmt.Print("Please enter your email: ")

	// Read the input (email address)
	email, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}

	// Remove the trailing newline character
	email = email[:len(email)-1]

	// generate minimal JSON config
	config := map[string]interface{}{
		"log": map[string]string{
			"level": logger.LevelInfoValue,
		},
		"acme": map[string]string{
			"email": email,
		},
	}

	location := filepath.Join(ConfigDirectory, "config.json")
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// write to file and change ownership
	file, err := os.Create(location)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	defer file.Close()
	_, err = file.Write(content)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
