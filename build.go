//go:build ignore

package main

import (
	"fmt"
	"minesweeper/misc"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"
)

const SettingsPath = "build-settings.txt"

var SettingsList []string
var DefaultSettings = make(map[string]bool)
var SettingsComments = make(map[string]string)

func init() {
	setDefault := func(name string, value bool, comment string) {
		SettingsList = append(SettingsList, name)
		DefaultSettings[name] = value
		SettingsComments[name] = comment
	}

	setDefault("always-draw", false, "Always redraw frames. Even if you are not doing anything.")
	setDefault("pprof", false, "Enable pporf debugging.")
	setDefault("dev", false, "Enable dev related features like debugging.")
	setDefault("opt", true, "Optimize and inline.")
	setDefault("wasm-opt", false, "Optimize wasm (requires wasm-opt from https://github.com/WebAssembly/binaryen).")
	setDefault("no-vcs", false, "Stop Go compiler from stamp binary with version control information.")
}

func PrintUsage() {
	scriptName := misc.GetScriptName()

	fmt.Printf("\n")
	fmt.Printf("Usage of %s:\n", scriptName)
	fmt.Printf("\n")
	fmt.Printf("go run %s\n", scriptName)
	fmt.Printf("go run %s [target]\n", scriptName)
	fmt.Printf("go run %s release [target]\n", scriptName)
	fmt.Printf("\n")
	fmt.Printf("valid targets:\n")
	fmt.Printf("  desktop\n")
	fmt.Printf("  web\n")
	fmt.Printf("  all\n")
	fmt.Printf("\n")
	fmt.Printf("build settings are read from %s\n", SettingsPath)
	fmt.Printf("\n")
	fmt.Printf("but if you do\n")
	fmt.Printf("go run %s release [target]\n", scriptName)
	fmt.Printf("it uses release settings\n")
	fmt.Printf("\n")
}

func main() {
	args := os.Args[1:]

	// print help
	{
		helps := []string{
			"help",
			"-help",
			"--help",
			"h",
			"-h",
			"--h",
		}
		if len(args) > 0 && slices.Contains(helps, args[0]) {
			PrintUsage()
			os.Exit(1)
		}
	}

	var buildTarget = "desktop"
	var useReleaseSetting = false

	// parse flags
	if len(args) == 1 {
		buildTarget = args[0]
	} else if len(args) == 2 {
		if args[0] != "release" {
			misc.ErrLogger.Printf("%s is not a vaid argument", strings.Join(args, " "))
			PrintUsage()
			os.Exit(1)
		} else {
			useReleaseSetting = true
		}
		buildTarget = args[1]
	} else if len(args) > 2 {
		misc.ErrLogger.Printf("too many arguments")
		PrintUsage()
		os.Exit(1)
	}

	if !(buildTarget == "desktop" || buildTarget == "web" || buildTarget == "all") {
		misc.ErrLogger.Printf("%s is not a vaid target", buildTarget)
		PrintUsage()
		os.Exit(1)
	}

	// if settings file doesn't exist, create one
	if exist, err := misc.CheckFileExists(SettingsPath); err != nil {
		misc.ErrLogger.Printf("could not check if %s file exists: %v", SettingsPath, err)
		os.Exit(1)
	} else if !exist {

		misc.InfoLogger.Printf("couldn't find %s, making a default one", SettingsPath)

		err := SaveSettings(SettingsPath, DefaultSettings)
		if err != nil {
			misc.ErrLogger.Printf("could not write default settings to %s: %v", SettingsPath, err)
			os.Exit(1)
		}
	}

	var settings map[string]bool

	if useReleaseSetting {
		// TODO: support release setting
		misc.ErrLogger.Printf("release setting is not supproted yet")
		os.Exit(1)
	} else {
		// load settings
		misc.InfoLogger.Printf("loading settings from %s", SettingsPath)
		var err error
		settings, err = LoadSettings(SettingsPath)

		if err != nil {
			misc.ErrLogger.Printf("failed to load settings : %v", err)
			os.Exit(1)
		}
	}

	// print settings
	{
		nameSize := 0
		for _, name := range SettingsList {
			nameSize = max(nameSize, len(name))
		}
		fmt.Printf("\n")
		for _, name := range SettingsList {
			value := settings[name]
			for len(name) < nameSize {
				name = name + " "
			}
			fmt.Printf("  %v : %v\n", name, value)
		}
		fmt.Printf("\n")
	}

	misc.InfoLogger.Printf("building %s", buildTarget)

	buildDesktop := func() {
		err, errcode := BuildApp(settings, false)
		if err != nil {
			misc.ErrLogger.Printf("failed to build for desktop: %v", err)
			os.Exit(errcode)
		}
	}
	buildWeb := func() {
		err, errcode := BuildApp(settings, true)
		if err != nil {
			misc.ErrLogger.Printf("failed to build for web: %v", err)
			os.Exit(errcode)
		}
	}

	switch buildTarget {
	case "desktop":
		buildDesktop()
	case "web":
		buildWeb()
	case "all":
		buildDesktop()
		buildWeb()
	}
}

func SetMissingSettingsToDefault(settings map[string]bool) {
	for _, name := range SettingsList {
		if _, ok := settings[name]; !ok {
			settings[name] = DefaultSettings[name]
		}
	}
}

func CopySettings(settings map[string]bool) map[string]bool {
	settingsCopy := make(map[string]bool)
	for k, v := range settings {
		settingsCopy[k] = v
	}

	SetMissingSettingsToDefault(settingsCopy)

	return settingsCopy
}

func SaveSettings(path string, settings map[string]bool) error {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "// settings file for building\n")
	fmt.Fprintf(sb, "// lines starting with // are comments\n")
	fmt.Fprintf(sb, "//\n")
	fmt.Fprintf(sb, "// comment/uncomment these settings\n")
	fmt.Fprintf(sb, "\n")
	fmt.Fprintf(sb, "\n")
	for _, settingName := range SettingsList {
		comment := SettingsComments[settingName]
		value := settings[settingName]

		fmt.Fprintf(sb, "// %s\n", comment)
		fmt.Fprintf(sb, "%s %v\n", settingName, value)
		fmt.Fprintf(sb, "\n")
	}
	err := os.WriteFile(path, []byte(sb.String()), 0664)
	if err != nil {
		return err
	}
	return nil
}

func LoadSettings(path string) (map[string]bool, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if !utf8.Valid(file) {
		return nil, fmt.Errorf("not a valid utf8 file")
	}

	text := string(file)
	text = strings.ReplaceAll(text, "\r\n", "\n")

	lines := strings.Split(text, "\n")

	settings := CopySettings(DefaultSettings)

	for i, line := range lines {
		logWarning := func(format string, a ...any) {
			fileAndLine := fmt.Sprintf("%s:%d: ", path, i+1)
			fmt.Fprintf(os.Stderr, fileAndLine+format+"\n", a...)
		}

		trimmed := strings.TrimSpace(line)

		if len(trimmed) <= 0 { // ignore empty line
			continue
		}

		if strings.HasPrefix(trimmed, "//") { // ignore comments
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			logWarning("\"%s\" doesn't have two fields, ignored", line)
			continue
		}

		if _, ok := DefaultSettings[fields[0]]; !ok {
			logWarning("\"%s\" is not a valid option, ignored", fields[0])
			continue
		}

		if fields[1] == "true" {
			settings[fields[0]] = true
		} else if fields[1] == "false" {
			settings[fields[0]] = false
		} else {
			logWarning("\"%s\" is not true or false, ignored", fields[1])
			continue
		}
	}

	SetMissingSettingsToDefault(settings)

	return settings, nil
}

func BuildApp(settings map[string]bool, buildWeb bool) (error, int) {
	tags := ""

	if settings["always-draw"] {
		tags += "alwaysdraw,"
	}
	if settings["pprof"] {
		tags += "minepprof,"
	}
	if settings["dev"] {
		tags += "minedev,"
		misc.WarnLogger.Printf("dev option is not implemented")
	}

	gcFlags := "-e -l -N"
	if settings["opt"] {
		gcFlags = "-e"
	}

	dst := "minesweeper"
	if runtime.GOOS == "windows" {
		dst += ".exe"
	}
	if buildWeb {
		dst = "./web_build/minesweeper.wasm"
	}

	cmd := exec.Command(
		"go",
		"build",
		"-o", dst,
		"-tags="+tags,
		"-gcflags=all="+gcFlags,
	)

	if settings["no-vcs"] {
		cmd.Args = append(cmd.Args, "-buildvcs=false")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if buildWeb {
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, "GOOS=js")
		cmd.Env = append(cmd.Env, "GOARCH=wasm")
	}

	misc.InfoLogger.Printf("%s", strings.Join(cmd.Args, " "))

	fmt.Printf("\n")
	err := cmd.Run()
	fmt.Printf("\n")

	if err != nil {
		return err, err.(*exec.ExitError).ExitCode()
	}

	if buildWeb && settings["wasm-opt"] {
		misc.InfoLogger.Printf("optimizing using wasm-opt")
		// check if wasm-opt exists
		if !misc.CheckExeExists("wasm-opt") {
			return fmt.Errorf("couldn't find wasm-opt"), 1
		}

		cmd := exec.Command(
			"wasm-opt",
			"./web_build/minesweeper.wasm",
			"-O2",
			"--enable-bulk-memory-opt", // NOTE: I have no idea what this option does lol
			"-o",
			"./web_build/minesweeper-opt.wasm",
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		misc.InfoLogger.Printf("%s", strings.Join(cmd.Args, " "))

		fmt.Printf("\n")
		err := cmd.Run()
		fmt.Printf("\n")

		if err != nil {
			return err, err.(*exec.ExitError).ExitCode()
		}

		err = os.Rename("./web_build/minesweeper.wasm", "./web_build/minesweeper.wasm.bak")
		if err != nil {
			return err, 1
		}
		err = os.Rename("./web_build/minesweeper-opt.wasm", "./web_build/minesweeper.wasm")
		if err != nil {
			return err, 1
		}
	}

	return nil, 0
}
