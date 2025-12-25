//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
)

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = RunWithLocalConfig

// A build step that requires additional params, or platform specific steps for example
func build() error {
	//mg.Deps(InstallDeps)
	fmt.Println("Building...")
	cmd := exec.Command("go", "build", "-o", "go-veeam-influx", ".")
	return cmd.Run()
}

// Runs main.go
func RunWithLocalConfig() error {
	cmd := exec.Command("go", "run", "main.go", "-config", "config-local.yaml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run with custom config
func RunWithConfig(configFile string) error {
	cmd := exec.Command("go", "run", "main.go", "-config", configFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Run with docker compose
func RunDocker() error {
	cmd := exec.Command("docker", "compose", "-f", "docker/docker-compose.yaml", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ExportConfig() error {
	cmd := exec.Command("go", "run", "main.go", "-export")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func BuildDocker(version string) error {
	cmd := exec.Command("docker", "build", "-t", fmt.Sprintf("govein:%s", version), "-f", "docker/Dockerfile", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// A custom install step if you need your bin someplace other than go/bin
func install() error {
	mg.Deps(build)
	fmt.Println("Installing...")
	return os.Rename("./MyApp", "/usr/bin/MyApp")
}

// Manage your deps, or running package managers.
func installDeps() error {
	fmt.Println("Installing Deps...")
	//cmd := exec.Command("go", "get", "github.com/stretchr/piglatin")
	//return cmd.Run()
	return nil
}

// Clean build artifacts
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("go-veeam-influx")
	cmd := exec.Command("docker", "compose", "-f", "docker/docker-compose.yaml", "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
