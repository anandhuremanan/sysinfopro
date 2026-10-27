package process

import (
	"os"
	"strings"
	"testing"
)

func TestRefuseReason(t *testing.T) {
	self := int32(9999)
	refused := []struct {
		pid  int32
		name string
	}{{0, "System Idle Process"}, {4, "System"}, {9999, "sysinfopro.exe"}, {612, "lsass.exe"}, {700, "CSRSS.EXE"}, {1200, "svchost.exe"}, {88, "Registry"}}
	for _, c := range refused {
		if refuseReason(c.pid, c.name, self) == "" {
			t.Errorf("%d %s should be refused", c.pid, c.name)
		}
	}
	for _, name := range []string{"node.exe", "chrome.exe", "Code.exe"} {
		if why := refuseReason(5000, name, self); why != "" {
			t.Errorf("%s should be allowed, got %q", name, why)
		}
	}
}

func TestTerminateRefusesSelfAndWrongName(t *testing.T) {
	if err := TerminateProcess(int32(os.Getpid())); err == nil || !strings.Contains(err.Error(), "SysInfo Pro itself") {
		t.Errorf("ending this process must be refused, got %v", err)
	}
	// the test binary's own PID under a wrong name: refused before any kill
	if err := TerminateProcessChecked(int32(os.Getpid()), "other.exe", 0); err == nil {
		t.Error("expected refusal")
	}
}
