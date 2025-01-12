package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	timeout int
	noColor bool
)

var AnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Show detailed analysis of current cluster context",
	Run: func(cmd *cobra.Command, args []string) {
		// Create a channel to signal when to stop the spinner
		stopSpinner := make(chan bool)
		spinnerDone := make(chan bool)

		// Start the spinner in a goroutine
		go func() {
			spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			i := 0
			for {
				select {
				case <-stopSpinner:
					fmt.Print("\r\033[K") // Clear the line
					spinnerDone <- true
					return
				default:
					fmt.Printf("\r%s Analyzing cluster...", spinner[i])
					i = (i + 1) % len(spinner)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()

		metrics, err := getClusterMetrics()

		// Stop the spinner and wait for cleanup
		stopSpinner <- true
		<-spinnerDone

		if err != nil {
			fmt.Printf("Error analyzing cluster: %v\n", err)
			return
		}

		printMetricsTable(metrics)
	},
}

func init() {
	AnalyzeCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "Timeout in seconds")
	AnalyzeCmd.Flags().BoolVarP(&noColor, "no-color", "n", false, "Disable color output")
}

type ClusterMetrics struct {
	Context          string
	TotalNamespaces  int
	TotalPods        int
	RunningPods      int
	CrashedPods      int
	PendingPods      int
	TotalNodes       int
	ReadyNodes       int
	NotReadyNodes    int
	TotalCPU         string
	TotalMemory      string
	UsedCPU          string
	UsedMemory       string
	TotalDeployments int
	TotalServices    int
	IngressCount     int
	PVCCount         int
	SecretCount      int
	ConfigMapCount   int
	NodeHealth       map[string]string
	TopPodsByCPU     []string
	TopPodsByMemory  []string
}

func getClusterMetrics() (*ClusterMetrics, error) {
	metrics := &ClusterMetrics{
		NodeHealth: make(map[string]string),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	type metricResult struct {
		metric string
		output []byte
		err    error
	}

	// Create buffered channels
	ch := make(chan metricResult, 15)
	done := make(chan bool)

	// Run commands concurrently with timeout
	var wg sync.WaitGroup
	commands := map[string]*exec.Cmd{
		"context":     exec.Command("kubectl", "config", "current-context"),
		"namespaces":  exec.Command("kubectl", "get", "namespaces", "--no-headers"),
		"pods":        exec.Command("kubectl", "get", "pods", "--all-namespaces", "--no-headers"),
		"nodes":       exec.Command("kubectl", "get", "nodes", "-o", "wide", "--no-headers"),
		"deployments": exec.Command("kubectl", "get", "deployments", "--all-namespaces", "--no-headers"),
		"services":    exec.Command("kubectl", "get", "services", "--all-namespaces", "--no-headers"),
		"ingress":     exec.Command("kubectl", "get", "ingress", "--all-namespaces", "--no-headers"),
		"pvc":         exec.Command("kubectl", "get", "pvc", "--all-namespaces", "--no-headers"),
		"secrets":     exec.Command("kubectl", "get", "secrets", "--all-namespaces", "--no-headers"),
		"configmaps":  exec.Command("kubectl", "get", "configmaps", "--all-namespaces", "--no-headers"),
		"top-nodes":   exec.Command("kubectl", "top", "nodes", "--no-headers"),
		"top-pods":    exec.Command("kubectl", "top", "pods", "--all-namespaces", "--no-headers", "--sort-by=cpu"),
	}

	for name, cmd := range commands {
		wg.Add(1)
		go func(name string, cmd *exec.Cmd) {
			defer wg.Done()
			output, err := cmd.Output()
			select {
			case <-ctx.Done():
				return
			case ch <- metricResult{name, output, err}:
			}
		}(name, cmd)
	}

	// Wait for all commands or timeout
	go func() {
		wg.Wait()
		close(ch)
		done <- true
	}()

	// Collect results with timeout
	results := make(map[string][]byte)
	failedCommands := make([]string, 0)

	for i := 0; i < len(commands); i++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation timed out after %d seconds", timeout)
		case result, ok := <-ch:
			if !ok {
				continue
			}
			if result.err != nil {
				failedCommands = append(failedCommands, result.metric)
			} else {
				results[result.metric] = result.output
			}
		}
	}

	// Process results
	if output, ok := results["context"]; ok {
		metrics.Context = strings.TrimSpace(string(output))
	}

	if output, ok := results["namespaces"]; ok {
		metrics.TotalNamespaces = len(strings.Split(strings.TrimSpace(string(output)), "\n"))
	}

	if output, ok := results["pods"]; ok {
		pods := strings.Split(strings.TrimSpace(string(output)), "\n")
		metrics.TotalPods = len(pods)
		for _, pod := range pods {
			if strings.Contains(pod, "Running") {
				metrics.RunningPods++
			} else if strings.Contains(pod, "Error") || strings.Contains(pod, "CrashLoopBackOff") {
				metrics.CrashedPods++
			} else if strings.Contains(pod, "Pending") {
				metrics.PendingPods++
			}
		}
	}

	if output, ok := results["nodes"]; ok {
		nodes := strings.Split(strings.TrimSpace(string(output)), "\n")
		metrics.TotalNodes = len(nodes)
		for _, node := range nodes {
			fields := strings.Fields(node)
			if len(fields) >= 2 {
				nodeName := fields[0]
				status := fields[1]
				metrics.NodeHealth[nodeName] = status
				if status == "Ready" {
					metrics.ReadyNodes++
				} else {
					metrics.NotReadyNodes++
				}
			}
		}
	}

	// Process resource counts
	resourceCounts := map[string]*int{
		"deployments": &metrics.TotalDeployments,
		"services":    &metrics.TotalServices,
		"ingress":     &metrics.IngressCount,
		"pvc":         &metrics.PVCCount,
		"secrets":     &metrics.SecretCount,
		"configmaps":  &metrics.ConfigMapCount,
	}

	for resource, count := range resourceCounts {
		if output, ok := results[resource]; ok {
			*count = len(strings.Split(strings.TrimSpace(string(output)), "\n"))
		}
	}

	// Process top metrics
	var totalCPU, totalMem float64
	var usedCPU, usedMem float64

	// Get all node capacities in one call
	nodeCmd := exec.Command("kubectl", "get", "nodes", "-o", `jsonpath={range .items[*]}{.status.capacity.cpu}{"\t"}{.status.capacity.memory}{"\n"}{end}`)
	if capacityOutput, err := nodeCmd.Output(); err == nil {
		capacities := strings.Split(strings.TrimSpace(string(capacityOutput)), "\n")
		for _, capacity := range capacities {
			fields := strings.Split(capacity, "\t")
			if len(fields) == 2 {
				cpuCores := parseFloat(fields[0])
				memKi := parseFloat(strings.TrimSuffix(fields[1], "Ki"))
				totalCPU += cpuCores
				totalMem += memKi / (1024 * 1024) // Convert Ki to GB
			}
		}
	}

	// Get usage percentages from top output
	if topOutput, ok := results["top-nodes"]; ok {
		lines := strings.Split(strings.TrimSpace(string(topOutput)), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				cpuUsage := strings.TrimSuffix(fields[2], "%")
				memUsage := strings.TrimSuffix(fields[4], "%")
				usedCPU += parseFloat(cpuUsage)
				usedMem += parseFloat(memUsage)
			}
		}
	}

	if metrics.TotalNodes > 0 {
		metrics.TotalCPU = fmt.Sprintf("%.0f cores", totalCPU)
		metrics.TotalMemory = fmt.Sprintf("%.0f GB", totalMem)
		metrics.UsedCPU = fmt.Sprintf("%.1f%%", usedCPU/float64(metrics.TotalNodes))
		metrics.UsedMemory = fmt.Sprintf("%.1f%%", usedMem/float64(metrics.TotalNodes))
	}

	// Process top pods
	if output, ok := results["top-pods"]; ok {
		pods := strings.Split(strings.TrimSpace(string(output)), "\n")
		for i := 0; i < 3 && i < len(pods); i++ {
			metrics.TopPodsByCPU = append(metrics.TopPodsByCPU, pods[i])
			metrics.TopPodsByMemory = append(metrics.TopPodsByMemory, pods[i])
		}
	}

	// Report any failed commands
	if len(failedCommands) > 0 {
		yellow := color.New(color.FgYellow).SprintFunc()
		fmt.Printf("\nWarning: Some metrics unavailable: %s\n", yellow(strings.Join(failedCommands, ", ")))
	}

	return metrics, nil
}

func printMetricsTable(metrics *ClusterMetrics) {
	red := color.New(color.FgRed).SprintFunc()

	// Helper function to print a complete table
	printTable := func(title string, rows [][2]string) {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', 0)

		// Calculate max width for value column
		maxWidth := 0
		for _, row := range rows {
			if len(row[1]) > maxWidth {
				maxWidth = len(row[1])
			}
		}
		if maxWidth < 20 {
			maxWidth = 20
		}

		// Create separator line
		separator := fmt.Sprintf("+%s+%s+",
			strings.Repeat("-", 15),
			strings.Repeat("-", maxWidth+2))

		// Print table
		fmt.Fprintln(w, separator)
		fmt.Fprintf(w, "| %-13s | %-*s |\n", title, maxWidth, "")
		fmt.Fprintln(w, separator)

		for _, row := range rows {
			// Handle colored output without breaking alignment
			value := row[1]
			padding := strings.Repeat(" ", maxWidth-len(strings.TrimSpace(value)))
			if strings.Contains(value, "\x1b") { // Check if value contains ANSI color codes
				fmt.Fprintf(w, "| %-13s | %s%s |\n", row[0], value, padding)
			} else {
				fmt.Fprintf(w, "| %-13s | %-*s |\n", row[0], maxWidth, value)
			}
		}
		fmt.Fprintln(w, separator)
		w.Flush()
		fmt.Println()
	}

	// Print top pods table
	printTopPods := func(pods []string) {
		if len(pods) == 0 {
			return
		}

		// Calculate maximum lengths
		maxNamespace := 16 // minimum width
		maxName := 40      // minimum width
		maxUsage := 11     // minimum width

		// First pass: determine required column widths
		for _, pod := range pods {
			fields := strings.Fields(pod)
			if len(fields) >= 4 {
				if len(fields[0]) > maxNamespace {
					maxNamespace = len(fields[0])
				}
				if len(fields[1]) > maxName {
					maxName = len(fields[1])
				}
				usage := strings.Join(fields[2:], " ")
				if len(usage) > maxUsage {
					maxUsage = len(usage)
				}
			}
		}

		// Add padding
		maxNamespace += 2
		maxName += 2
		maxUsage += 2

		// Create the separator line
		separator := fmt.Sprintf("+%s+%s+%s+",
			strings.Repeat("-", maxNamespace),
			strings.Repeat("-", maxName),
			strings.Repeat("-", maxUsage))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', 0)
		fmt.Fprintln(w, separator)
		fmt.Fprintf(w, "| %-*s | %-*s | %-*s |\n",
			maxNamespace-2, "NAMESPACE",
			maxName-2, "NAME",
			maxUsage-2, "CPU/MEM")
		fmt.Fprintln(w, separator)

		for _, pod := range pods {
			fields := strings.Fields(pod)
			if len(fields) >= 4 {
				namespace := fields[0]
				name := fields[1]
				usage := strings.Join(fields[2:], " ")

				// Truncate with ellipsis if too long
				if len(name) > maxName-5 {
					name = name[:maxName-6] + "..."
				}
				if len(namespace) > maxNamespace-5 {
					namespace = namespace[:maxNamespace-6] + "..."
				}

				fmt.Fprintf(w, "| %-*s | %-*s | %*s |\n",
					maxNamespace-2, namespace,
					maxName-2, name,
					maxUsage-2, usage)
			}
		}
		fmt.Fprintln(w, separator)
		w.Flush()
		fmt.Println()
	}

	// Context
	printTable("CONTEXT", [][2]string{
		{"Current", metrics.Context},
	})

	// Nodes
	rows := [][2]string{
		{"Total", fmt.Sprintf("%d", metrics.TotalNodes)},
		{"Ready", fmt.Sprintf("%d", metrics.ReadyNodes)},
	}
	if !noColor && metrics.NotReadyNodes > 0 {
		rows = append(rows, [2]string{"Not Ready", red(fmt.Sprintf("%d", metrics.NotReadyNodes))})
	} else {
		rows = append(rows, [2]string{"Not Ready", fmt.Sprintf("%d", metrics.NotReadyNodes)})
	}
	printTable("NODES", rows)

	// Workloads
	printTable("WORKLOADS", [][2]string{
		{"Namespaces", fmt.Sprintf("%d", metrics.TotalNamespaces)},
		{"Deployments", fmt.Sprintf("%d", metrics.TotalDeployments)},
		{"Services", fmt.Sprintf("%d", metrics.TotalServices)},
		{"Ingresses", fmt.Sprintf("%d", metrics.IngressCount)},
	})

	// Pods
	rows = [][2]string{
		{"Total", fmt.Sprintf("%d", metrics.TotalPods)},
		{"Running", fmt.Sprintf("%d", metrics.RunningPods)},
		{"Crashed", fmt.Sprintf("%d", metrics.CrashedPods)},
		{"Pending", fmt.Sprintf("%d", metrics.PendingPods)},
	}

	// Apply color to crashed pods if needed
	if !noColor && metrics.CrashedPods > 0 {
		rows[2][1] = red(fmt.Sprintf("%d", metrics.CrashedPods))
	}
	printTable("PODS", rows)

	// Storage
	printTable("STORAGE", [][2]string{
		{"PVCs", fmt.Sprintf("%d", metrics.PVCCount)},
		{"Secrets", fmt.Sprintf("%d", metrics.SecretCount)},
		{"ConfigMaps", fmt.Sprintf("%d", metrics.ConfigMapCount)},
	})

	// Resources
	if metrics.TotalCPU != "" {
		printTable("RESOURCES", [][2]string{
			{"CPU Total", metrics.TotalCPU},
			{"CPU Used", metrics.UsedCPU},
			{"Memory Total", metrics.TotalMemory},
			{"Memory Used", metrics.UsedMemory},
		})
	}

	// Top Pods
	if len(metrics.TopPodsByCPU) > 0 {
		printTopPods(metrics.TopPodsByCPU)
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
