package apparmor

import (
	"os"
	"sync"
)

var (
	appArmorEnabled bool
	checkAppArmor   sync.Once
)

// IsEnabled returns true if apparmor is enabled on this host
func IsEnabled() bool {
	checkAppArmor.Do(func() {
		appArmorEnabled = checkAppArmorStatus()
	})
	return appArmorEnabled
}

func checkAppArmorStatus() bool {
    // Check if AppArmor is mounted
    if _, err := os.Stat("/sys/kernel/security/apparmor"); err != nil {
        return false
    }

    // Check if AppArmor module is enabled
    buf, err := os.ReadFile("/sys/module/apparmor/parameters/enabled")
    if err != nil {
        return false
    }

    return len(buf) > 1 && buf[0] == 'Y'
}
