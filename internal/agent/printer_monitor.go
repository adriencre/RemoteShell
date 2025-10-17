package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"remoteshell/internal/common"
)

// PrinterMonitor gère le monitoring des imprimantes
type PrinterMonitor struct {
	lastUpdate time.Time
	cache      map[string]*common.PrinterInfo
}

// NewPrinterMonitor crée un nouveau moniteur d'imprimantes
func NewPrinterMonitor() *PrinterMonitor {
	return &PrinterMonitor{
		cache: make(map[string]*common.PrinterInfo),
	}
}

// GetPrinters retourne la liste des imprimantes
func (pm *PrinterMonitor) GetPrinters() ([]*common.PrinterInfo, error) {
	if runtime.GOOS == "windows" {
		return pm.getWindowsPrinters()
	}
	return pm.getLinuxPrinters()
}

// getLinuxPrinters récupère les imprimantes via CUPS
func (pm *PrinterMonitor) getLinuxPrinters() ([]*common.PrinterInfo, error) {
	var printers []*common.PrinterInfo

	// Utiliser lpstat pour lister les imprimantes
	cmd := exec.Command("lpstat", "-p", "-d")
	output, err := cmd.Output()
	if err != nil {
		// Si CUPS n'est pas disponible, essayer d'autres méthodes
		return pm.getLinuxPrintersAlternative()
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var currentPrinter *common.PrinterInfo

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Ligne d'imprimante par défaut
		if strings.HasPrefix(line, "system default destination:") {
			defaultName := strings.TrimSpace(strings.TrimPrefix(line, "system default destination:"))
			if printer, exists := pm.cache[defaultName]; exists {
				printer.IsDefault = true
			}
			continue
		}

		// Ligne d'imprimante
		if strings.HasPrefix(line, "printer ") {
			// Sauvegarder l'imprimante précédente
			if currentPrinter != nil {
				pm.cache[currentPrinter.Name] = currentPrinter
				printers = append(printers, currentPrinter)
			}

			// Parser la ligne d'imprimante
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[1]
				status := "unknown"
				if len(parts) >= 4 {
					status = strings.Trim(parts[3], "()")
				}

				currentPrinter = &common.PrinterInfo{
					Name:   name,
					Status: status,
					Jobs:   []common.PrintJob{},
				}
			}
		}
	}

	// Ajouter la dernière imprimante
	if currentPrinter != nil {
		pm.cache[currentPrinter.Name] = currentPrinter
		printers = append(printers, currentPrinter)
	}

	// Récupérer les détails des imprimantes
	for _, printer := range printers {
		pm.getLinuxPrinterDetails(printer)
		pm.getLinuxPrinterJobs(printer)
	}

	pm.lastUpdate = time.Now()
	return printers, nil
}

// getLinuxPrinterDetails récupère les détails d'une imprimante Linux
func (pm *PrinterMonitor) getLinuxPrinterDetails(printer *common.PrinterInfo) {
	// Utiliser lpoptions pour récupérer les détails
	cmd := exec.Command("lpoptions", "-p", printer.Name, "-l")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "printer-info") {
			// Extraire la description
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				printer.Description = strings.Join(parts[1:], " ")
			}
		}
		if strings.Contains(line, "printer-location") {
			// Extraire la localisation
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				printer.Location = strings.Join(parts[1:], " ")
			}
		}
		if strings.Contains(line, "device-uri") {
			// Extraire l'URI
			parts := strings.Split(line, " ")
			if len(parts) > 1 {
				printer.URI = strings.Join(parts[1:], " ")
			}
		}
	}
}

// getLinuxPrinterJobs récupère les travaux d'impression d'une imprimante Linux
func (pm *PrinterMonitor) getLinuxPrinterJobs(printer *common.PrinterInfo) {
	cmd := exec.Command("lpq", "-P", printer.Name)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	// Ignorer la première ligne (header)
	if scanner.Scan() {
		scanner.Text()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		job := pm.parseLinuxJobLine(line)
		if job != nil {
			printer.Jobs = append(printer.Jobs, *job)
		}
	}
}

// parseLinuxJobLine parse une ligne de travail d'impression Linux
func (pm *PrinterMonitor) parseLinuxJobLine(line string) *common.PrintJob {
	// Format typique: "Rank   Owner   Job     File(s)                         Total Size"
	// Exemple: "1st    user    123     document.pdf                     1024 bytes"
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil
	}

	job := &common.PrintJob{
		Status: "active",
	}

	// Parser l'ID du job
	if id, err := strconv.Atoi(parts[2]); err == nil {
		job.ID = id
	}

	// Parser le nom du fichier
	if len(parts) > 3 {
		job.Name = parts[3]
	}

	// Parser l'utilisateur
	if len(parts) > 1 {
		job.User = parts[1]
	}

	// Parser la taille si disponible
	if len(parts) > 4 {
		sizeStr := strings.Join(parts[4:], " ")
		job.Size = pm.parseSize(sizeStr)
	}

	job.Created = time.Now() // Approximation
	return job
}

// getLinuxPrintersAlternative méthode alternative pour Linux sans CUPS
func (pm *PrinterMonitor) getLinuxPrintersAlternative() ([]*common.PrinterInfo, error) {
	var printers []*common.PrinterInfo

	// Essayer /etc/printcap
	if data, err := exec.Command("cat", "/etc/printcap").Output(); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Parser une ligne de printcap
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				name := strings.TrimSpace(parts[0])
				printer := &common.PrinterInfo{
					Name:        name,
					Status:      "unknown",
					Description: "Legacy printer",
					Jobs:        []common.PrintJob{},
				}
				printers = append(printers, printer)
			}
		}
	}

	return printers, nil
}

// getWindowsPrinters récupère les imprimantes via WMI
func (pm *PrinterMonitor) getWindowsPrinters() ([]*common.PrinterInfo, error) {
	var printers []*common.PrinterInfo

	// Utiliser PowerShell pour interroger WMI
	psScript := `
	Get-WmiObject -Class Win32_Printer | ForEach-Object {
		$printer = @{
			Name = $_.Name
			Status = $_.PrinterStatus
			Description = $_.Description
			Location = $_.Location
			PortName = $_.PortName
			Default = $_.Default
		}
		$printer | ConvertTo-Json -Compress
	}
	`

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("erreur PowerShell: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var printerData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &printerData); err != nil {
			continue
		}

		printer := &common.PrinterInfo{
			Name:        getString(printerData, "Name"),
			Status:      pm.mapWindowsStatus(getInt(printerData, "Status")),
			Description: getString(printerData, "Description"),
			Location:    getString(printerData, "Location"),
			URI:         getString(printerData, "PortName"),
			IsDefault:   getBool(printerData, "Default"),
			Jobs:        []common.PrintJob{},
		}

		printers = append(printers, printer)
	}

	// Récupérer les travaux d'impression
	for _, printer := range printers {
		pm.getWindowsPrinterJobs(printer)
	}

	pm.lastUpdate = time.Now()
	return printers, nil
}

// getWindowsPrinterJobs récupère les travaux d'impression Windows
func (pm *PrinterMonitor) getWindowsPrinterJobs(printer *common.PrinterInfo) {
	psScript := fmt.Sprintf(`
		Get-WmiObject -Class Win32_PrintJob | Where-Object { $_.Name -like "*%s*" } | ForEach-Object {
			$job = @{
				ID = $_.JobId
				Name = $_.Document
				User = $_.Owner
				Status = $_.Status
				Size = $_.Size
			}
			$job | ConvertTo-Json -Compress
		}
	`, printer.Name)

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var jobData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jobData); err != nil {
			continue
		}

		job := common.PrintJob{
			ID:       getInt(jobData, "ID"),
			Name:     getString(jobData, "Name"),
			User:     getString(jobData, "User"),
			Status:   getString(jobData, "Status"),
			Size:     int64(getInt(jobData, "Size")),
			Created:  time.Now(), // Approximation
		}

		printer.Jobs = append(printer.Jobs, job)
	}
}

// mapWindowsStatus convertit le statut Windows en texte lisible
func (pm *PrinterMonitor) mapWindowsStatus(status int) string {
	statusMap := map[int]string{
		1:  "autre",
		2:  "inconnu",
		3:  "idle",
		4:  "printing",
		5:  "warmup",
	}
	
	if mapped, exists := statusMap[status]; exists {
		return mapped
	}
	return "unknown"
}

// parseSize parse une taille en bytes
func (pm *PrinterMonitor) parseSize(sizeStr string) int64 {
	// Regex pour extraire le nombre et l'unité
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(\w*)`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) < 2 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToLower(matches[2])
	switch unit {
	case "kb", "k":
		return int64(value * 1024)
	case "mb", "m":
		return int64(value * 1024 * 1024)
	case "gb", "g":
		return int64(value * 1024 * 1024 * 1024)
	default:
		return int64(value) // Assume bytes
	}
}

// Fonctions utilitaires pour l'extraction de données JSON
func getString(data map[string]interface{}, key string) string {
	if val, exists := data[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if val, exists := data[key]; exists {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

func getBool(data map[string]interface{}, key string) bool {
	if val, exists := data[key]; exists {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}


