package hatchcert

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const Day = time.Hour * 24

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	d = d.Truncate(time.Minute)
	days := int(d / Day)
	if days > 3 {
		return fmt.Sprint(days, "d")
	}
	remaining := d - time.Duration(days)*Day
	hours := int(remaining.Hours())
	if days > 1 {
		if hours == 0 {
			return fmt.Sprint(days, "d")
		}
		return fmt.Sprint(days, "d", hours, "h")
	}
	minutes := int(remaining.Minutes()) % 60
	if hours == 0 {
		return fmt.Sprint(minutes, "m")
	}
	if minutes == 0 {
		return fmt.Sprint(hours, "h")
	}
	return fmt.Sprint(hours, "h", minutes, "m")
}

func unmarshal(fname string, v interface{}) error {
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func marshal(fname string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fname, b, 0644)
}

func exists(fname string) bool {
	_, err := os.Stat(fname)
	return err == nil
}

// TODO: this sucks
func replaceLink(dir, target, name string) error {
	r, err := rand.Int(rand.Reader, big.NewInt(99999))
	if err != nil {
		return err
	}

	// Make random symlink pointing at destination
	temp := filepath.Join(dir, fmt.Sprint(r, "--", name))
	if err := os.Symlink(target, temp); err != nil {
		return err
	}

	// Replace existing symlink
	dst := filepath.Join(dir, name)
	err = os.Rename(temp, dst)
	if err == nil {
		// This can fail if e.g. the target is a directory; there should be
		// some recovery logic for this
		os.Remove(temp)
	}
	return err
}

func safePath(fname string) bool {
	for _, s := range fname {
		if s <= ' ' || s == '\\' || s == '/' || s == ':' {
			return false
		}
	}
	return fname != "" && fname != "." && fname != ".."
}
