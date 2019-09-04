// +build !windows,!darwin

package volume

import (
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var useAmixer bool

func init() {
	if _, err := exec.LookPath("pactl"); err != nil {
		useAmixer = true
	}
}

func cmdEnv() []string {
	return []string{"LANG=C", "LC_ALL=C"}
}

func getVolumeCmd() []string {
	if useAmixer {
		return []string{"amixer", "get", "Master"}
	}
	return []string{"pactl", "list", "sinks"}
}

func getPulseAudioCmd() []string {
	return []string{"pacmd", "list"}
}

func getFallbackSink() (int, error) {
	out, err := execCmd(getPulseAudioCmd())
	if err != nil {
		return 0, err
	}
	pacmd := string(out)
	lines := strings.Split(pacmd, "\n")
	for _, line := range lines {
		if !useAmixer && strings.Contains(line, "*") {
			return strconv.Atoi(line[len(line)-1:])
		}
	}
	return 0, errors.New("no fallback sink found")
}

var volumePattern = regexp.MustCompile(`\d+%`)

func parseVolume(fbSink int, out string) (int, error) {
	sinks := strings.Split(out, "\n\n")
	for _, sink := range sinks {
		s := strings.TrimLeft(sink, " \t")
		sinkLines := strings.Split(s, "\n")
		if sinkLines[0] == "Sink #"+strconv.Itoa(fbSink) {
			for _, line := range sinkLines {
				s := strings.TrimLeft(line, " \t")
				if !useAmixer && strings.HasPrefix(s, "State:") {

				}
				if useAmixer && strings.Contains(s, "Playback") && strings.Contains(s, "%") ||
					!useAmixer && strings.HasPrefix(s, "Volume:") {
					volumeStr := volumePattern.FindString(s)
					return strconv.Atoi(volumeStr[:len(volumeStr)-1])
				}
			}
		}
	}

	return 0, errors.New("no volume found")
}

func setVolumeCmd(fbSink int, volume int) []string {
	if useAmixer {
		return []string{"amixer", "set", "Master", strconv.Itoa(volume) + "%"}
	}
	return []string{"pactl", "set-sink-volume", strconv.Itoa(fbSink), strconv.Itoa(volume) + "%"}
}

func increaseVolumeCmd(fbSink int, diff int) []string {
	var sign string
	if diff >= 0 {
		sign = "+"
	} else if useAmixer {
		diff = -diff
		sign = "-"
	}
	if useAmixer {
		return []string{"amixer", "set", "Master", strconv.Itoa(diff) + "%" + sign}
	}
	return []string{"pactl", "--", "set-sink-volume", strconv.Itoa(fbSink), sign + strconv.Itoa(diff) + "%"}
}

func getMutedCmd() []string {
	if useAmixer {
		return []string{"amixer", "get", "Master"}
	}
	return []string{"pactl", "list", "sinks"}
}

func parseMuted(fbSink int, out string) (bool, error) {
	sinks := strings.Split(out, "\n\n")
	for _, sink := range sinks {
		s := strings.TrimLeft(sink, " \t")
		sinkLines := strings.Split(s, "\n")
		if sinkLines[0] == "Sink #"+strconv.Itoa(fbSink) {
			for _, line := range sinkLines {
				s := strings.TrimLeft(line, " \t")
				if useAmixer && strings.Contains(s, "Playback") && strings.Contains(s, "%") ||
					!useAmixer && strings.HasPrefix(s, "Mute: ") {
					if strings.Contains(s, "[off]") || strings.Contains(s, "yes") {
						return true, nil
					} else if strings.Contains(s, "[on]") || strings.Contains(s, "no") {
						return false, nil
					}
				}
			}
		}
	}
	return false, errors.New("no muted information found")
}

func muteCmd(fbSink int) []string {
	if useAmixer {
		return []string{"amixer", "-D", "pulse", "set", "Master", "mute"}
	}
	return []string{"pactl", "set-sink-mute", strconv.Itoa(fbSink), "1"}
}

func unmuteCmd(fbSink int) []string {
	if useAmixer {
		return []string{"amixer", "-D", "pulse", "set", "Master", "unmute"}
	}
	return []string{"pactl", "set-sink-mute", strconv.Itoa(fbSink), "0"}
}
