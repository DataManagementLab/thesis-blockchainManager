package docker

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
)

func CreateVolume(volumeName string, verbose bool) error {
	return RunDockerCommand(".", verbose, verbose, "volume", "create", volumeName)
}
func CreateNetwork(networkName string, verbose bool) error{
	return RunDockerCommand(".", verbose, verbose, "network", "create", networkName)
}

func CopyFileToVolume(volumeName string, sourcePath string, destPath string, verbose bool) error {
	fileName := path.Base(sourcePath)
	return RunDockerCommand(".", verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/source/%s", sourcePath, fileName), "-v", fmt.Sprintf("%s:/dest", volumeName), "alpine", "cp", path.Join("/", "source", fileName), path.Join("/", "dest", destPath))
}

func MkdirInVolume(volumeName string, directory string, verbose bool) error {
	return RunDockerCommand(".", verbose, verbose, "run", "--rm", "-v", fmt.Sprintf("%s:/dest", volumeName), "alpine", "mkdir", "-p", path.Join("/", "dest", directory))
}

func RemoveVolume(volumeName string, verbose bool) error {
	return RunDockerCommand(".", verbose, verbose, "volume", "remove", volumeName)
}

func CopyFromContainer(containerName string, sourcePath string, destPath string, verbose bool) error {
	if err := RunDockerCommand(".", verbose, verbose, "cp", containerName+":"+sourcePath, destPath); err != nil {
		return err
	}
	return nil
}

func RunDockerCommandRetry(workingDir string, showCommand bool, pipeStdout bool, retries int, command ...string) error {
	attempt := 0
	for {
		err := RunDockerCommand(workingDir, showCommand, pipeStdout, command...)
		if err != nil && attempt < retries {
			attempt++
			continue
		} else if err != nil {
			return err
		}
		break
	}
	return nil
}

func RunDockerCommand(workingDir string, showCommand bool, pipeStdout bool, command ...string) error {
	dockerCmd := exec.Command("docker", command...)
	dockerCmd.Dir = workingDir
	_, err := runCommand(dockerCmd, showCommand, pipeStdout, command...)
	return err
}

func RunDockerComposeCommand(workingDir string, showCommand bool, pipeStdout bool, command ...string) error {
	dockerCmd := exec.Command("docker-compose", command...)
	dockerCmd.Dir = workingDir
	_, err := runCommand(dockerCmd, showCommand, pipeStdout, command...)
	return err
}
func DockerReturnCommand(workingDir string, showCommand bool, pipeStdout bool, command ...string) (string, error) {
	dockerCmd := exec.Command("docker", command...)
	dockerCmd.Dir = workingDir
	return dockerCmd.String(), nil
}

func RunDockerCommandBuffered(workingDir string, showCommand bool, command ...string) (string, error) {
	dockerCmd := exec.Command("docker", command...)
	dockerCmd.Dir = workingDir
	return runCommand(dockerCmd, showCommand, false, command...)
}

func runCommand(cmd *exec.Cmd, showCommand bool, pipeStdout bool, command ...string) (string, error) {
	if showCommand {
		fmt.Println(cmd.String())
	}
	outputBuff := strings.Builder{}
	stdoutChan := make(chan string)
	stderrChan := make(chan string)
	errChan := make(chan error)
	go pipeCommand(cmd, stdoutChan, stderrChan, errChan)

outputCapture:
	for {
		select {
		case s, ok := <-stdoutChan:
			if pipeStdout {
				if !ok {
					break outputCapture
				}
				fmt.Print(s)
			} else {
				outputBuff.WriteString(s)
			}
		case s, ok := <-stderrChan:
			if !ok {
				break outputCapture
			}
			if pipeStdout {
				fmt.Print(s)
			} else {
				outputBuff.WriteString(s)
			}
		case err := <-errChan:
			return "", err
		}
	}
	cmd.Wait()
	statusCode := cmd.ProcessState.ExitCode()
	if statusCode != 0 {
		return "", fmt.Errorf("%s\nFailed [%d] %s", strings.Join(cmd.Args, " "), statusCode, outputBuff.String())
	}
	return outputBuff.String(), nil
}

func pipeCommand(cmd *exec.Cmd, stdoutChan chan string, stderrChan chan string, errChan chan error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errChan <- err
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		errChan <- err
		return
	}
	cmd.Start()
	go readPipe(stdout, stdoutChan, errChan)
	go readPipe(stderr, stderrChan, errChan)
}

func readPipe(pipe io.ReadCloser, outputChan chan string, errChan chan error) {
	buf := bufio.NewReader(pipe)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				close(outputChan)
				return
			} else {
				errChan <- err
				close(outputChan)
				return
			}
		} else {
			outputChan <- line
		}
	}
}
