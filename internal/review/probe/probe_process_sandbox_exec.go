package probe

func buildProbeProcessSandboxExec(cmd probeExecCommand) (probeExecCommand, error) {
	if !cmd.sandbox.enabled {
		return cmd, nil
	}

	args, err := cmd.sandbox.buildBubblewrapArgs(cmd)
	if err != nil {
		return probeExecCommand{}, err
	}

	wrapped := cmd
	wrapped.commandPath = cmd.sandbox.runnerPath
	wrapped.args = args
	return wrapped, nil
}

func (s probeProcessSandbox) buildBubblewrapArgs(cmd probeExecCommand) ([]string, error) {
	binds, err := s.bubblewrapBindArgs(cmd)
	if err != nil {
		return nil, err
	}

	args := []string{
		"--unshare-all",
		// 一部 CI では network namespace 内の loopback 設定が拒否される。
		// probe の安全境界は filesystem sandbox なので、network は host と共有する。
		"--share-net",
		"--die-with-parent",
		"--new-session",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--dir", "/var",
		"--tmpfs", "/var/tmp",
	}
	args = append(args, binds...)
	args = append(args, "--clearenv")
	args = append(args, bubblewrapEnvArgs(cmd.env)...)
	args = append(args, "--chdir", cmd.workDir)
	args = append(args, cmd.commandPath)
	args = append(args, cmd.args...)
	return args, nil
}

func bubblewrapEnvArgs(env []string) []string {
	args := make([]string, 0, len(env)*3)
	for _, entry := range env {
		key, value, ok := splitEnvEntry(entry)
		if !ok {
			continue
		}
		args = append(args, "--setenv", key, value)
	}
	return args
}
