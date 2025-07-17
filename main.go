package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port    int
	Name    string
	Verbose bool
}

type RequestInfo struct {
	Hostname     string            `json:"hostname"`
	Name         string            `json:"name,omitempty"`
	IP           []string          `json:"ip"`
	RemoteAddr   string            `json:"remote_addr"`
	Host         string            `json:"host"`
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	RealIP       string            `json:"real_ip"`
	Protocol     string            `json:"protocol"`
	Headers      map[string]string `json:"headers"`
	Environment  map[string]string `json:"environment"`
	OS           string            `json:"os"`
	Architecture string            `json:"architecture"`
	Runtime      string            `json:"runtime"`
	Time         string            `json:"time"`
	Version      string            `json:"version"`
}

var (
	config  Config
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func init() {
	flag.IntVar(&config.Port, "port", getEnvAsInt("WHOAMI_PORT_NUMBER", 80), "The port number")
	flag.StringVar(&config.Name, "name", os.Getenv("WHOAMI_NAME"), "The name")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getLocalIPs() []string {
	var ips []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}

func getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func getRequestInfo(r *http.Request) *RequestInfo {
	hostname, _ := os.Hostname()

	// Get all headers
	headers := make(map[string]string)
	for name, values := range r.Header {
		headers[name] = strings.Join(values, ", ")
	}

	// Get environment variables
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if pair := strings.SplitN(e, "=", 2); len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}

	return &RequestInfo{
		Hostname:     hostname,
		Name:         config.Name,
		IP:           getLocalIPs(),
		RemoteAddr:   r.RemoteAddr,
		Host:         r.Host,
		URL:          r.URL.String(),
		Method:       r.Method,
		RealIP:       getRealIP(r),
		Protocol:     r.Proto,
		Headers:      headers,
		Environment:  env,
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Runtime:      runtime.Version(),
		Time:         time.Now().Format(time.RFC3339),
		Version:      version,
	}
}

func formatAsText(info *RequestInfo) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Hostname: %s\n", info.Hostname))
	if info.Name != "" {
		sb.WriteString(fmt.Sprintf("Name: %s\n", info.Name))
	}
	sb.WriteString(fmt.Sprintf("IP: %s\n", strings.Join(info.IP, ", ")))
	sb.WriteString(fmt.Sprintf("RemoteAddr: %s\n", info.RemoteAddr))
	sb.WriteString(fmt.Sprintf("Host: %s\n", info.Host))
	sb.WriteString(fmt.Sprintf("URL: %s\n", info.URL))
	sb.WriteString(fmt.Sprintf("Method: %s\n", info.Method))
	sb.WriteString(fmt.Sprintf("RealIP: %s\n", info.RealIP))
	sb.WriteString(fmt.Sprintf("Protocol: %s\n", info.Protocol))
	sb.WriteString(fmt.Sprintf("OS: %s\n", info.OS))
	sb.WriteString(fmt.Sprintf("Architecture: %s\n", info.Architecture))
	sb.WriteString(fmt.Sprintf("Runtime: %s\n", info.Runtime))
	sb.WriteString(fmt.Sprintf("Time: %s\n", info.Time))
	sb.WriteString(fmt.Sprintf("Version: %s\n", info.Version))

	sb.WriteString("\nHeaders:\n")
	var headerKeys []string
	for key := range info.Headers {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)
	for _, key := range headerKeys {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", key, info.Headers[key]))
	}

	sb.WriteString("\nEnvironment:\n")
	var envKeys []string
	for key := range info.Environment {
		envKeys = append(envKeys, key)
	}
	sort.Strings(envKeys)
	for _, key := range envKeys {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", key, info.Environment[key]))
	}

	return sb.String()
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if config.Verbose {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, getRealIP(r))
	}

	info := getRequestInfo(r)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, formatAsText(info))
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	if config.Verbose {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, getRealIP(r))
	}

	info := getRequestInfo(r)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(info); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		if config.Verbose {
			log.Printf("JSON encoding error: %v", err)
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if config.Verbose {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, getRealIP(r))
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"version": version,
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Add version flag
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("whoami %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
		fmt.Printf("  built by: %s\n", builtBy)
		os.Exit(0)
	}

	if config.Verbose {
		log.Printf("Starting whoami server on port %d", config.Port)
		log.Printf("Name: %s", config.Name)
		log.Printf("Verbose logging enabled")
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/api", apiHandler)
	http.HandleFunc("/health", healthHandler)

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Server listening on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
