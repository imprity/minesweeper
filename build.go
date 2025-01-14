//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"minesweeper/misc"
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
	setDefault("tsc", false, "Build typescript module.")
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

	// build tsc
	{
		tscTargets := []string{
			"./web_build/sound.js",
		}
		tscSources := []string{
			"./sound/sound.ts",
			"./sound/internalplayer.ts",

			"./sound/package-lock.json",
			"./sound/package.json",
			"./sound/tsconfig.json",
		}

		if settings["tsc"] && NeedToBuild(tscTargets, tscSources) {
			err, errcode := BuildTsc()
			if err != nil {
				misc.ErrLogger.Printf("failed to build for typescript module: %v", err)
				os.Exit(errcode)
			}
		}
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
	// generate sound_srcs.go
	{
		var targets = []string{
			"sound_srcs.go",
			"sound_srcs_bin",
		}

		var srcs = []string{
			"sound_srcs_gen.go",
		}

		const soundSrcsTxt = "sound_srcs.txt"
		const soundSrcsLocalTxt = "sound_srcs_local.txt"

		if exists, err := misc.CheckFileExists(soundSrcsTxt); err != nil {
			return err, 1
		} else if exists {
			srcs = append(srcs, soundSrcsTxt)
		}
		if exists, err := misc.CheckFileExists(soundSrcsLocalTxt); err != nil {
			return err, 1
		} else if exists {
			srcs = append(srcs, soundSrcsLocalTxt)
		}

		if NeedToBuild(targets, srcs) {
			cmd := exec.Command(
				"go", "run", "sound_srcs_gen.go",
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			misc.InfoLogger.Printf("%s", cmd.String())

			fmt.Printf("\n")
			err := cmd.Run()
			fmt.Printf("\n")

			if err != nil {
				return err, err.(*exec.ExitError).ExitCode()
			}
		}
	}

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

	misc.InfoLogger.Printf("%s", cmd.String())

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

		misc.InfoLogger.Printf("%s", cmd.String())

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

func BuildTsc() (error, int) {
	if !misc.CheckExeExists("npm") {
		return fmt.Errorf("couldn't find npm"), 1
	}

	// ==============
	// npm install
	// ==============
	cmd := exec.Command(
		"npm",
		"install",
	)

	cmd.Dir = "./sound"

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	misc.InfoLogger.Printf("%s", cmd.String())

	err := cmd.Run()

	if err != nil {
		return err, err.(*exec.ExitError).ExitCode()
	}

	// ================
	// run typescript
	// ================
	if !misc.CheckExeExists("npx") {
		return fmt.Errorf("couldn't find npx"), 1
	}

	cmd = exec.Command(
		"npx",
		"tsc",
	)

	cmd.Dir = "./sound"

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	misc.InfoLogger.Printf("%s", cmd.String())

	fmt.Printf("\n")
	err = cmd.Run()
	fmt.Printf("\n")

	if err != nil {
		return err, err.(*exec.ExitError).ExitCode()
	}

	// ================
	// copy file
	// ================
	const copySrc = "./sound/sound.js"
	const copyDst = "./web_build/sound.js"

	misc.InfoLogger.Printf("copying %s to %s", copySrc, copyDst)

	err = misc.CopyFile(
		copySrc, copyDst, 0664,
	)
	if err != nil {
		return err, 1
	}

	return nil, 0
}

func NeedToBuild(targets []string, srcs []string) bool {
	// if any of the targets don't exist,
	// we definitely need to build it
	for _, target := range targets {
		if exists, err := misc.CheckFileExists(target); err != nil {
			misc.ErrLogger.Fatalf("failed to check if %s exists: %v", target, err)
		} else if !exists {
			return true
		}
	}

	var srcNewest time.Time
	var targetOldest time.Time

	var srcNewestSet bool = false
	var targetOldestSet bool = false

	for _, src := range srcs {
		info, err := os.Stat(src)
		if err != nil {
			misc.ErrLogger.Fatalf("failed to check mod time of %s: %v", src, err)
		}
		modTime := info.ModTime()

		if !srcNewestSet {
			srcNewest = modTime
			srcNewestSet = true
		} else if srcNewest.Compare(modTime) < 0 {
			srcNewest = modTime
		}
	}

	for _, target := range targets {
		info, err := os.Stat(target)
		if err != nil {
			misc.ErrLogger.Fatalf("failed to check mod time of %s: %v", target, err)
		}
		modTime := info.ModTime()

		if !targetOldestSet {
			targetOldest = modTime
			targetOldestSet = true
		} else if targetOldest.Compare(modTime) > 0 {
			targetOldest = modTime
		}
	}

	return srcNewest.Compare(targetOldest) > 0
}
