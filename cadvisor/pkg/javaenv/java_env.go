// Package javaenv reads only WHATAP_JAVA_AGENT_PATH from a container's
// startup-resolved runtime configuration. It never looks up Kubernetes secrets
// or ConfigMaps, logs environment data, or caches failed observations.
package javaenv

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const inspectTimeout = 3 * time.Second
const maxRuntimeConfigBytes = 4 << 20

func fromEnv(env []string) string {
	const prefix = "WHATAP_JAVA_AGENT_PATH="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return strings.TrimPrefix(env[i], prefix)
		}
	}
	return ""
}

func fromSpec(spec *specs.Spec) string {
	if spec == nil || spec.Process == nil {
		return ""
	}
	return fromEnv(spec.Process.Env)
}

// Inspect uses a short-lived, deadline-bounded client so a not-yet-ready runtime
// cannot poison a shared client/cache. Runtime errors are deliberately sanitized:
// upstream error text may contain a response body with unrelated environment.
func Inspect(runtime, containerID string) (string, error) {
	if containerID == "" {
		return "", errors.New("missing container ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), inspectTimeout)
	defer cancel()
	var value string
	var err error
	switch runtime {
	case "containerd":
		value, err = inspectContainerd(ctx, containerID)
	case "docker":
		var cli *client.Client
		cli, err = client.NewClientWithOpts(client.WithVersion("1.40"))
		if err == nil {
			defer cli.Close()
			value, err = inspectDocker(ctx, cli, containerID)
		}
	case "crio":
		value, err = inspectCrio("/rootfs", containerID)
	default:
		return "", errors.New("no supported container runtime detected")
	}
	if err != nil {
		return "", errors.New("Java agent runtime environment unavailable")
	}
	return value, nil
}

func inspectContainerd(ctx context.Context, id string) (string, error) {
	cli, err := containerd.New("/run/containerd/containerd.sock", containerd.WithTimeout(inspectTimeout))
	if err != nil {
		return "", err
	}
	defer cli.Close()
	nss, err := cli.NamespaceService().List(ctx)
	if err != nil {
		return "", err
	}
	for _, ns := range nss {
		nsctx := namespaces.WithNamespace(ctx, ns)
		container, err := cli.LoadContainer(nsctx, id)
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			continue
		}
		spec, err := container.Spec(nsctx)
		if err != nil {
			return "", err
		}
		return fromSpec(spec), nil
	}
	return "", errors.New("container runtime config not found")
}

func inspectDocker(ctx context.Context, cli *client.Client, id string) (string, error) {
	info, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}
	if info.Config == nil {
		return "", nil
	}
	return fromEnv(info.Config.Env), nil
}

func inspectCrio(root, id string) (string, error) {
	path := filepath.Join(root, "var/lib/containers/storage/overlay-containers", id, "userdata", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if alt := os.Getenv("overlay_config_path"); alt != "" {
			path = filepath.Join(alt, id, "userdata", "config.json")
		}
	}
	// Check before opening: opening a FIFO for reading can itself block.
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !stat.Mode().IsRegular() || stat.Size() > maxRuntimeConfigBytes {
		return "", errors.New("invalid runtime config file")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	stat, err = file.Stat()
	if err != nil {
		return "", err
	}
	if !stat.Mode().IsRegular() || stat.Size() > maxRuntimeConfigBytes {
		return "", errors.New("invalid runtime config file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRuntimeConfigBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxRuntimeConfigBytes {
		return "", errors.New("runtime config too large")
	}
	var spec specs.Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return "", err
	}
	return fromSpec(&spec), nil
}
