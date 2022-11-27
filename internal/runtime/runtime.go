package runtime

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Environment interface {
	Get() map[string]string
	WorkingDir() string
}

type EmptyEnv struct{}

func (e *EmptyEnv) Get() map[string]string {
	return make(map[string]string)
}

func (e *EmptyEnv) WorkingDir() string {
	return ""
}

func Exec(env Environment, command string, stdOut io.Writer) error {

	vars := env.Get()
	runCmd := ""
	for k, v := range vars {
		runCmd = runCmd + fmt.Sprintf("export %s=\"%s\" && ", k, v)
	}
	runCmd = runCmd + command

	fmt.Println(runCmd)

	cmd := exec.Command("/bin/sh", "-c", runCmd)
	fmt.Println(env.WorkingDir())
	if env.WorkingDir() != "" {
		cmd.Dir = env.WorkingDir()
	}
	cmd.Stdout = stdOut
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}
