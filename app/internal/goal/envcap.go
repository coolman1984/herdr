package goal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Capability report (docs/COMPANY-ENVIRONMENT-POLICY.md): discover first, then plan. Every fact has a
// status, the source it came from and when it was seen. A probe that fails is "unknown", never "absent"
// and never a guessed number.

// Fact statuses.
const (
	CapTested   = "present-tested"   // ran and answered
	CapUntested = "present-untested" // found, not exercised
	CapAbsent   = "absent"
	CapDenied   = "denied" // present but refused by policy or permissions
	CapUnknown  = "unknown"
)

type Fact struct {
	Value  string `json:"value,omitempty"`
	Status string `json:"status"`
	Source string `json:"source"`
}

type DevicePolicy struct {
	// Profile: "company_nasca" (corporate PC, NASCA-protected files: Office COM only, no package
	// downloads unless stated) or "personal". Set by the owner per device; never guessed from files.
	Profile        string            `json:"profile"`
	AllowPackages  *bool             `json:"allow_packages,omitempty"`   // nil = unknown: treated as not allowed
	PackageAttempt map[string]string `json:"package_attempts,omitempty"` // package -> result of the one bounded try
}

type CapReport struct {
	Observed  time.Time       `json:"observed"`
	Host      map[string]Fact `json:"host"`
	Resources map[string]Fact `json:"resources"`
	Tools     map[string]Fact `json:"tools"`
	Office    map[string]Fact `json:"office"`
	Network   map[string]Fact `json:"network"`
	Policy    DevicePolicy    `json:"policy"`
	Budget    Budget          `json:"budget"`
	Unknowns  []string        `json:"unknowns"`
}

// Budget is derived from measured resources, not from the owner's PC.
type Budget struct {
	FreeRAMMB int     `json:"free_ram_mb"` // container limit applied
	CPUs      float64 `json:"cpus"`        // cgroup quota applied
	MaxAgents int     `json:"max_agents"`  // delegates; 0 = defer launching
	MaxProcs  int     `json:"max_procs"`   // lead + delegates + jobs
	BatchRows int     `json:"batch_rows"`  // document/data pipeline batch size
	Reason    string  `json:"reason"`
}

// Budget knobs: what each process is expected to take. Measured values replace these when known.
var (
	RAMReserveMB  = 1536 // left for the OS and the owner
	RAMPerAgentMB = 600
	RAMLeadMB     = 700
	RAMJobMB      = 400
)

func fact(v, status, src string) Fact { return Fact{Value: v, Status: status, Source: src} }

// toolVersion runs `name args` with a short timeout: present-tested with its first line, absent when
// not on PATH, unknown when it is there but did not answer.
func toolVersion(name string, args ...string) Fact {
	p, err := exec.LookPath(name)
	if err != nil {
		return fact("", CapAbsent, "PATH")
	}
	out, err, timedOut := runBounded(context.Background(), "", 10*time.Second, 4<<10, p, args...)
	line := strings.TrimSpace(strings.SplitN(strings.TrimSpace(out), "\n", 2)[0])
	if err != nil || timedOut || line == "" {
		return fact(p, CapUnknown, name+" "+strings.Join(args, " ")+" did not answer")
	}
	return fact(line, CapTested, p)
}

func readInt(path string) (int64, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(b))
	if s == "max" {
		return -1, true
	}
	n, err := strconv.ParseInt(s, 10, 64)
	return n, err == nil
}

// Discover builds the report. dir is the project (for disk). policyPath is the device policy file.
func Discover(dir, policyPath string) CapReport {
	r := CapReport{Observed: time.Now(), Host: map[string]Fact{}, Resources: map[string]Fact{}, Tools: map[string]Fact{},
		Office: map[string]Fact{}, Network: map[string]Fact{}}
	r.Host["os"] = fact(runtime.GOOS+"/"+runtime.GOARCH, CapTested, "runtime")
	if h, err := os.Hostname(); err == nil {
		r.Host["hostname"] = fact(h, CapTested, "os.Hostname")
	}
	if u := os.Getenv(map[bool]string{true: "USERNAME", false: "USER"}[runtime.GOOS == "windows"]); u != "" {
		r.Host["user"] = fact(u, CapTested, "env")
	} else {
		r.Host["user"] = fact("", CapUnknown, "USER/USERNAME not set")
	}
	if runtime.GOOS == "windows" {
		sess := os.Getenv("SESSIONNAME")
		st := CapTested
		if sess == "" {
			st = CapUnknown // a service or scheduled task: Office COM may not have a desktop
		}
		r.Host["session"] = fact(sess, st, "SESSIONNAME (Console = interactive desktop)")
	} else {
		r.Host["uid"] = fact(strconv.Itoa(os.Getuid()), CapTested, "getuid")
		container := ""
		if _, err := os.Stat("/.dockerenv"); err == nil {
			container = "docker"
		}
		if b, err := os.ReadFile("/proc/1/cgroup"); err == nil && (strings.Contains(string(b), "kubepods") || strings.Contains(string(b), "docker")) {
			container = "container (cgroup)"
		}
		if os.Getenv("container") != "" {
			container = os.Getenv("container")
		}
		if container == "" {
			if b, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil && strings.Contains(strings.ToLower(string(b)), "microsoft") {
				container = "wsl"
			}
		}
		if container != "" {
			r.Host["container"] = fact(container, CapTested, "/.dockerenv, /proc/1/cgroup, $container")
		} else {
			r.Host["container"] = fact("none seen", CapUntested, "/.dockerenv, /proc/1/cgroup")
		}
	}

	// Resources: host numbers, then the container's own limits, which win.
	free := FreeRAMMB()
	if free >= 0 {
		r.Resources["ram_available_mb"] = fact(strconv.Itoa(free), CapTested, "MemAvailable / GlobalMemoryStatusEx")
	} else {
		r.Resources["ram_available_mb"] = fact("", CapUnknown, "no memory probe")
	}
	if lim, ok := readInt("/sys/fs/cgroup/memory.max"); ok && lim > 0 {
		used, _ := readInt("/sys/fs/cgroup/memory.current")
		left := int((lim - used) >> 20)
		r.Resources["cgroup_memory_left_mb"] = fact(strconv.Itoa(left), CapTested, "/sys/fs/cgroup/memory.max - memory.current")
		if free < 0 || left < free {
			free = left
		}
	} else if ok {
		r.Resources["cgroup_memory_left_mb"] = fact("no limit", CapTested, "/sys/fs/cgroup/memory.max")
	}
	cpus := float64(runtime.NumCPU())
	r.Resources["cpus_visible"] = fact(strconv.Itoa(runtime.NumCPU()), CapTested, "runtime.NumCPU")
	if b, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		f := strings.Fields(string(b))
		if len(f) == 2 && f[0] != "max" {
			q, _ := strconv.ParseFloat(f[0], 64)
			p, _ := strconv.ParseFloat(f[1], 64)
			if p > 0 && q/p < cpus {
				cpus = q / p
			}
			r.Resources["cgroup_cpus"] = fact(fmt.Sprintf("%.2f", q/p), CapTested, "/sys/fs/cgroup/cpu.max")
		}
	}
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		r.Resources["load_1m"] = fact(strings.Fields(string(b))[0], CapTested, "/proc/loadavg")
	}
	if d := FreeDiskMB(dir); d >= 0 {
		r.Resources["disk_free_mb"] = fact(strconv.Itoa(d), CapTested, "statfs/GetDiskFreeSpaceEx "+dir)
	} else {
		r.Resources["disk_free_mb"] = fact("", CapUnknown, dir)
	}

	// Tools, PowerShell first on Windows (whatever version exists).
	if runtime.GOOS == "windows" {
		r.Tools["pwsh"] = toolVersion("pwsh", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()")
		r.Tools["powershell"] = toolVersion("powershell", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()")
		r.Tools["py"] = toolVersion("py", "--version")
	} else {
		r.Tools["pwsh"] = toolVersion("pwsh", "-NoProfile", "-Command", "$PSVersionTable.PSVersion.ToString()")
		r.Tools["sh"] = toolVersion("sh", "-c", "echo sh")
	}
	for _, t := range [][]string{{"git", "--version"}, {"python3", "--version"}, {"python", "--version"}, {"node", "--version"}, {"go", "version"},
		{"sqlite3", "--version"}, {"claude", "--version"}, {"codex", "--version"}, {"agy", "--version"}, {"cline", "--version"}, {"herdr", "--version"}, {"soffice", "--version"}} {
		r.Tools[t[0]] = toolVersion(t[0], t[1:]...)
	}
	if py := pythonExe(r.Tools); py != "" {
		out, err, _ := runBounded(context.Background(), "", 15*time.Second, 4<<10, py, "-c",
			"import importlib.util as u,sqlite3,json;print(json.dumps({m:bool(u.find_spec(m)) for m in ['openpyxl','docx','pptx','pypdf','pdfplumber','win32com','duckdb','pyarrow']}|{'sqlite':sqlite3.sqlite_version}))")
		if err == nil {
			r.Tools["python_modules"] = fact(strings.TrimSpace(out), CapTested, py+" find_spec (no import, no install)")
		} else {
			r.Tools["python_modules"] = fact("", CapUnknown, py)
		}
	}

	// Office / COM: Windows registry only (reading a ProgID's CLSID never starts Office).
	if runtime.GOOS == "windows" {
		for _, app := range []string{"Excel.Application", "Word.Application", "PowerPoint.Application", "AcroExch.App"} {
			out, err, _ := runBounded(context.Background(), "", 10*time.Second, 4<<10, "reg", "query", `HKCR\`+app+`\CLSID`)
			if err == nil && strings.Contains(out, "{") {
				r.Office[app] = fact(regexp.MustCompile(`\{[0-9A-Fa-f-]+\}`).FindString(out), CapUntested, `HKCR\`+app+`\CLSID (registered; not started)`)
			} else {
				r.Office[app] = fact("", CapAbsent, `HKCR\`+app+`\CLSID`)
			}
		}
		out, err, _ := runBounded(context.Background(), "", 10*time.Second, 4<<10, "reg", "query", `HKLM\SOFTWARE\Microsoft\Office\ClickToRun\Configuration`, "/v", "Platform")
		if err == nil {
			r.Office["bitness"] = fact(strings.TrimSpace(out[strings.LastIndex(out, "REG_SZ")+6:]), CapTested, "ClickToRun Platform")
		} else {
			r.Office["bitness"] = fact("", CapUnknown, "ClickToRun Platform not readable (MSI install or no Office)")
		}
	} else {
		r.Office["com"] = fact("", CapAbsent, "COM exists only on Windows")
	}

	// Network: proxy settings and a bounded DNS lookup; no data is sent anywhere.
	for _, k := range []string{"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY"} {
		if v := os.Getenv(k); v != "" {
			r.Network[strings.ToLower(k)] = fact("set", CapTested, "env "+k)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if _, err := net.DefaultResolver.LookupHost(ctx, "pypi.org"); err == nil {
		r.Network["dns_pypi"] = fact("resolves", CapTested, "DNS lookup pypi.org")
	} else {
		r.Network["dns_pypi"] = fact(err.Error(), CapUnknown, "DNS lookup pypi.org")
	}
	cancel()

	// Device policy (owner's per-device decision).
	r.Policy = DevicePolicy{Profile: "unknown"}
	if b, err := os.ReadFile(policyPath); err == nil {
		json.Unmarshal(b, &r.Policy)
	}
	r.Budget = DeriveBudget(free, cpus, 2)
	for _, m := range []map[string]Fact{r.Host, r.Resources, r.Tools, r.Office, r.Network} {
		for k, f := range m {
			if f.Status == CapUnknown {
				r.Unknowns = append(r.Unknowns, k+": "+f.Source)
			}
		}
	}
	return r
}

func pythonExe(t map[string]Fact) string {
	for _, k := range []string{"python3", "python", "py"} {
		if f, ok := t[k]; ok && f.Status == CapTested {
			return k
		}
	}
	return ""
}

// DeriveBudget: delegates the measured machine can carry after reserving RAM for the OS, the lead and
// one background job; capped by CPUs and the configured ceiling. Unknown RAM => one agent (low trust).
func DeriveBudget(freeMB int, cpus float64, ceiling int) Budget {
	b := Budget{FreeRAMMB: freeMB, CPUs: cpus}
	if freeMB < 0 {
		b.MaxAgents, b.Reason = 1, "free RAM unknown: one agent until measured"
	} else {
		left := freeMB - RAMReserveMB - RAMLeadMB - RAMJobMB
		b.MaxAgents = max(0, left/RAMPerAgentMB)
		b.Reason = fmt.Sprintf("(%d MB free - %d reserve - %d lead - %d job) / %d per agent", freeMB, RAMReserveMB, RAMLeadMB, RAMJobMB, RAMPerAgentMB)
	}
	if byCPU := int(cpus / 1.5); byCPU < b.MaxAgents {
		b.MaxAgents = max(byCPU, min(1, b.MaxAgents))
		b.Reason += fmt.Sprintf("; CPU cap %.1f cpus", cpus)
	}
	if b.MaxAgents > ceiling {
		b.MaxAgents = ceiling
		b.Reason += fmt.Sprintf("; ceiling %d", ceiling)
	}
	if b.MaxAgents == 0 {
		b.Reason += "; too little memory: defer launching agents"
	}
	b.MaxProcs = b.MaxAgents + 2 // lead + one background job
	b.BatchRows = 5000
	if freeMB >= 0 {
		b.BatchRows = max(500, min(50000, (freeMB-RAMReserveMB)*20)) // ~50 KB per row in flight, bounded
	}
	return b
}

// WriteReport saves the report as JSON next to the ledger (or wherever asked).
func WriteReport(r CapReport, path string) error {
	b, _ := json.MarshalIndent(r, "", "  ")
	os.MkdirAll(filepath.Dir(path), 0o755)
	return writeAtomic(path, b)
}
