package agent

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"remoteshell/internal/common"
)

// ServiceManager gère les services systemd et Docker
type ServiceManager struct {
	hasSystemd bool
	hasDocker  bool
}

// NewServiceManager crée un nouveau gestionnaire de services
func NewServiceManager() *ServiceManager {
	sm := &ServiceManager{
		hasSystemd: checkSystemd(),
		hasDocker:  checkDocker(),
	}

	log.Printf("ServiceManager initialisé - systemd: %v, docker: %v", sm.hasSystemd, sm.hasDocker)
	return sm
}

// checkSystemd vérifie si systemd est disponible
func checkSystemd() bool {
	cmd := exec.Command("systemctl", "--version")
	err := cmd.Run()
	return err == nil
}

// checkDocker vérifie si Docker est disponible
func checkDocker() bool {
	cmd := exec.Command("docker", "--version")
	err := cmd.Run()
	return err == nil
}

// ListServices liste tous les services disponibles
func (sm *ServiceManager) ListServices() ([]*common.ServiceInfo, error) {
	var services []*common.ServiceInfo

	// Lister les services systemd
	if sm.hasSystemd {
		systemdServices, err := sm.listSystemdServices()
		if err != nil {
			log.Printf("Erreur lors de la liste des services systemd: %v", err)
		} else {
			services = append(services, systemdServices...)
		}
	}

	// Lister les conteneurs Docker
	if sm.hasDocker {
		dockerServices, err := sm.listDockerContainers()
		if err != nil {
			log.Printf("Erreur lors de la liste des conteneurs Docker: %v", err)
		} else {
			services = append(services, dockerServices...)
		}
	}

	return services, nil
}

// listSystemdServices liste les services systemd
func (sm *ServiceManager) listSystemdServices() ([]*common.ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lister les services avec systemctl
	cmd := exec.CommandContext(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("échec de la commande systemctl: %v", err)
	}

	var services []*common.ServiceInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: UNIT LOAD ACTIVE SUB DESCRIPTION
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		serviceName := fields[0]
		// load := fields[1]
		activeState := fields[2]
		subState := fields[3]
		description := ""
		if len(fields) > 4 {
			description = strings.Join(fields[4:], " ")
		}

		// Vérifier si le service est activé
		enabled := sm.isServiceEnabled(serviceName)

		service := &common.ServiceInfo{
			Name:        serviceName,
			Type:        "systemd",
			Status:      subState,
			State:       activeState,
			Description: description,
			Enabled:     enabled,
		}

		services = append(services, service)
	}

	return services, nil
}

// isServiceEnabled vérifie si un service systemd est activé
func (sm *ServiceManager) isServiceEnabled(serviceName string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "is-enabled", serviceName)
	output, _ := cmd.Output()
	status := strings.TrimSpace(string(output))
	return status == "enabled"
}

// listDockerContainers liste les conteneurs Docker
func (sm *ServiceManager) listDockerContainers() ([]*common.ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Lister tous les conteneurs (running et stopped)
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.ID}}|{{.Names}}|{{.Status}}|{{.Image}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("échec de la commande docker: %v", err)
	}

	var services []*common.ServiceInfo
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		containerID := parts[0]
		name := parts[1]
		status := parts[2]
		image := parts[3]

		// Déterminer l'état
		state := "inactive"
		if strings.HasPrefix(status, "Up") {
			state = "active"
		} else if strings.HasPrefix(status, "Exited") {
			state = "inactive"
		}

		service := &common.ServiceInfo{
			Name:        name,
			Type:        "docker",
			Status:      status,
			State:       state,
			Description: fmt.Sprintf("Container: %s", image),
			ContainerID: containerID,
			Image:       image,
		}

		services = append(services, service)
	}

	return services, nil
}

// GetServiceStatus obtient le statut d'un service spécifique
func (sm *ServiceManager) GetServiceStatus(name, serviceType string) (*common.ServiceInfo, error) {
	if serviceType == "systemd" && sm.hasSystemd {
		return sm.getSystemdStatus(name)
	} else if serviceType == "docker" && sm.hasDocker {
		return sm.getDockerStatus(name)
	}

	return nil, fmt.Errorf("type de service non supporté ou non disponible: %s", serviceType)
}

// getSystemdStatus obtient le statut d'un service systemd
func (sm *ServiceManager) getSystemdStatus(serviceName string) (*common.ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "systemctl", "show", serviceName, "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("échec de récupération du statut: %v", err)
	}

	info := &common.ServiceInfo{
		Name: serviceName,
		Type: "systemd",
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "ActiveState":
			info.State = value
		case "SubState":
			info.Status = value
		case "Description":
			info.Description = value
		case "UnitFileState":
			info.Enabled = (value == "enabled")
		}
	}

	return info, nil
}

// getDockerStatus obtient le statut d'un conteneur Docker
func (sm *ServiceManager) getDockerStatus(containerName string) (*common.ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Id}}|{{.Name}}|{{.State.Status}}|{{.Config.Image}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("échec de récupération du statut: %v", err)
	}

	line := strings.TrimSpace(string(output))
	parts := strings.Split(line, "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("format de sortie invalide")
	}

	containerID := parts[0]
	name := strings.TrimPrefix(parts[1], "/")
	status := parts[2]
	image := parts[3]

	state := "inactive"
	if status == "running" {
		state = "active"
	}

	return &common.ServiceInfo{
		Name:        name,
		Type:        "docker",
		Status:      status,
		State:       state,
		Description: fmt.Sprintf("Container: %s", image),
		ContainerID: containerID,
		Image:       image,
	}, nil
}

// ExecuteAction exécute une action sur un service
func (sm *ServiceManager) ExecuteAction(action *common.ServiceAction) (*common.ServiceResult, error) {
	result := &common.ServiceResult{
		Name:   action.Name,
		Type:   action.Type,
		Action: action.Action,
	}

	if action.Type == "systemd" && sm.hasSystemd {
		err := sm.executeSystemdAction(action)
		if err != nil {
			result.Success = false
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = fmt.Sprintf("Action '%s' exécutée avec succès", action.Action)
		}
	} else if action.Type == "docker" && sm.hasDocker {
		err := sm.executeDockerAction(action)
		if err != nil {
			result.Success = false
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = fmt.Sprintf("Action '%s' exécutée avec succès", action.Action)
		}
	} else {
		result.Success = false
		result.Message = fmt.Sprintf("Type de service non supporté: %s", action.Type)
	}

	return result, nil
}

// executeSystemdAction exécute une action systemd
func (sm *ServiceManager) executeSystemdAction(action *common.ServiceAction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch action.Action {
	case "start":
		cmd = exec.CommandContext(ctx, "systemctl", "start", action.Name)
	case "stop":
		cmd = exec.CommandContext(ctx, "systemctl", "stop", action.Name)
	case "restart":
		cmd = exec.CommandContext(ctx, "systemctl", "restart", action.Name)
	case "enable":
		cmd = exec.CommandContext(ctx, "systemctl", "enable", action.Name)
	case "disable":
		cmd = exec.CommandContext(ctx, "systemctl", "disable", action.Name)
	default:
		return fmt.Errorf("action non supportée: %s", action.Action)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("échec de l'action: %v - %s", err, stderr.String())
	}

	return nil
}

// executeDockerAction exécute une action Docker
func (sm *ServiceManager) executeDockerAction(action *common.ServiceAction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch action.Action {
	case "start":
		cmd = exec.CommandContext(ctx, "docker", "start", action.Name)
	case "stop":
		cmd = exec.CommandContext(ctx, "docker", "stop", action.Name)
	case "restart":
		cmd = exec.CommandContext(ctx, "docker", "restart", action.Name)
	default:
		return fmt.Errorf("action non supportée pour Docker: %s", action.Action)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("échec de l'action: %v - %s", err, stderr.String())
	}

	return nil
}
