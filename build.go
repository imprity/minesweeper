//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

var ReleaseSettings = map[string]bool{
	"always-draw": false,
	"screenshot":  false,
	"pprof":       false,
	"dev":         false,
	"opt":         true,
	"no-debug":    true,
	"wasm-opt":    true,
	"tsc":         true,
	"no-vcs":      false,
}

func init() {
	setDefault := func(name string, value bool, comment string) {
		SettingsList = append(SettingsList, name)
		DefaultSettings[name] = value
		SettingsComments[name] = comment
	}

	setDefault("always-draw", false, "Always redraw frames. Even if you are not doing anything.")
	setDefault("screenshot", false, "Enable screenshot.")
	setDefault("pprof", false, "Enable pporf debugging.")
	setDefault("dev", false, "Enable dev related features like debugging.")
	setDefault("opt", true, "Optimize and inline.")
	setDefault("no-debug", false, "Don't include debugging informations.")
	setDefault("wasm-opt", false, "Optimize wasm (requires wasm-opt from https://github.com/WebAssembly/binaryen).")
	setDefault("tsc", false, "Build typescript module.")
	setDefault("no-vcs", false, "Stop Go compiler from stamp binary with version control information.")

	// =======================================
	// check if ReleaseSettings are correct
	// =======================================

	// has all the default settings
	for key := range DefaultSettings {
		if _, ok := ReleaseSettings[key]; !ok {
			misc.ErrLogger.Fatalf("ReleaseSettings lacks setting for %s", key)
		}
	}

	// doesn't have anything that DefaultSettings lacks
	for key := range ReleaseSettings {
		if _, ok := DefaultSettings[key]; !ok {
			misc.ErrLogger.Fatalf("ReleaseSettings has unknonw setting %s", key)
		}
	}
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
	fmt.Printf("  itch\n")
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
	var isRelease = false

	// parse flags
	if len(args) == 1 {
		if args[0] == "release" {
			isRelease = true
		} else {
			buildTarget = args[0]
		}
	} else if len(args) == 2 {
		if args[0] != "release" {
			misc.ErrLogger.Printf("%s is not a vaid argument", strings.Join(args, " "))
			PrintUsage()
			os.Exit(1)
		} else {
			isRelease = true
		}
		buildTarget = args[1]
	} else if len(args) > 2 {
		misc.ErrLogger.Printf("too many arguments")
		PrintUsage()
		os.Exit(1)
	}

	if !(buildTarget == "desktop" || buildTarget == "web" || buildTarget == "itch" || buildTarget == "all") {
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

	if isRelease {
		misc.InfoLogger.Print("using release settings")
		settings = ReleaseSettings
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
	if settings["tsc"] {
		soundTargets := []string{
			"./web_build/sound.js",
		}
		soundSources := []string{
			"./sound/sound.ts",
			"./sound/internalplayer.ts",

			"./sound/package-lock.json",
			"./sound/package.json",
			"./sound/tsconfig.json",
		}

		if NeedToBuild(soundTargets, soundSources) {
			err, errcode := BuildSoundTsc()
			if err != nil {
				misc.ErrLogger.Printf("failed to build sound typescript module: %v", err)
				os.Exit(errcode)
			}
		}

		webPageTargets := []string{
			"./web_build/loader.js",
		}
		webPageSources := []string{
			"./web_build/scripts/loader.ts",

			"./web_build/scripts/package-lock.json",
			"./web_build/scripts/package.json",
			"./web_build/scripts/tsconfig.json",
		}

		if NeedToBuild(webPageTargets, webPageSources) {
			err, errcode := BuildWebPageTsc()
			if err != nil {
				misc.ErrLogger.Printf("failed to build web page typescript module: %v", err)
				os.Exit(errcode)
			}
		}
	}

	misc.InfoLogger.Printf("building %s", buildTarget)

	buildDesktop := func() {
		err, errcode := BuildApp(settings, isRelease, false)
		if err != nil {
			misc.ErrLogger.Printf("failed to build for desktop: %v", err)
			os.Exit(errcode)
		}
	}
	buildWeb := func() {
		err, errcode := BuildApp(settings, isRelease, true)
		if err != nil {
			misc.ErrLogger.Printf("failed to build for web: %v", err)
			os.Exit(errcode)
		}
	}
	buildItch := func() {
		err, errcode := BuildItch(settings, isRelease)
		if err != nil {
			misc.ErrLogger.Printf("failed to build itch: %v", err)
			os.Exit(errcode)
		}
	}

	switch buildTarget {
	case "desktop":
		buildDesktop()
	case "web":
		buildWeb()
	case "itch":
		buildItch()
	case "all":
		buildDesktop()
		buildWeb()
		buildItch()
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

func Generate(buildRelease bool) (error, int) {
	// generate git_version.txt
	{
		versionStr, err, exitCode := GetGitVersionString(buildRelease)

		if err != nil {
			if !buildRelease {
				misc.WarnLogger.Printf("failed to get git version : %v", err)
				versionStr = "unknonw"
			} else { // release build should not omit version string
				return err, exitCode
			}
		}

		misc.InfoLogger.Printf("git version : %s", versionStr)

		if err = os.WriteFile("git_version.txt", []byte(versionStr), 0644); err != nil {
			return err, 1
		}
	}

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

	return nil, 0
}

func BuildApp(
	settings map[string]bool,
	buildRelease bool,
	buildWeb bool,
) (error, int) {
	if err, exitCode := Generate(buildRelease); err != nil {
		return err, exitCode
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
	}
	if settings["screenshot"] {
		tags += "screenshot,"
	}

	gcFlags := "-e -l -N"
	if settings["opt"] {
		gcFlags = "-e"
	}

	dst := "minesweeper"
	dst = AddExeIfWindows(dst)
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

	if settings["no-debug"] {
		cmd.Args = append(cmd.Args, "-trimpath")
		cmd.Args = append(cmd.Args, "-ldflags=-s -w")
	}

	cmd.Args = append(cmd.Args, "main.go")

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
			"--enable-bulk-memory", // NOTE: I have no idea what this option does lol
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

	// output compiled binary size to minesweeper_wasm_size.js
	if buildWeb {
		misc.InfoLogger.Print(
			"ouputing size of ./web_build/minesweeper.wasm to ./web_build/minesweeper_wasm_size.js",
		)

		info, err := os.Stat("./web_build/minesweeper.wasm")
		if err != nil {
			return err, 1
		}

		size := info.Size()
		err = os.WriteFile(
			"./web_build/minesweeper_wasm_size.js",
			[]byte(fmt.Sprintf("const MINESWEEPER_WASM_SIZE  = %d", size)),
			0664,
		)
		if err != nil {
			return err, 1
		}
	}

	// create release build
	if buildRelease {
		if buildWeb {
			err := os.RemoveAll("./release/web")
			if err != nil {
				return err, 1
			}
			err = misc.MkDir("./release/web")
			if err != nil {
				return err, 1
			}
			files, err := GlobMultiple(
				[]string{
					"./web_build/*.html",
					"./web_build/*.js",
					"./web_build/*.wasm",
				},
			)
			if err != nil {
				return err, 1
			}
			err = ZipFiles(
				files,
				"./web_build",
				"./release/web/minesweeper.zip",
			)
			if err != nil {
				return err, 1
			}
		} else {
			err := os.RemoveAll("./release/desktop")
			if err != nil {
				return err, 1
			}
			err = misc.MkDir("./release/desktop")
			if err != nil {
				return err, 1
			}
			err = misc.CopyFile(
				AddExeIfWindows("./minesweeper"),
				AddExeIfWindows("./release/desktop/minesweeper"),
				0755,
			)
			if err != nil {
				return err, 1
			}
		}
	}

	return nil, 0
}

func BuildItch(
	settings map[string]bool,
	buildRelease bool,
) (error, int) {
	if err, exitCode := Generate(buildRelease); err != nil {
		return err, exitCode
	}

	cmd := exec.Command(
		"go",
		"build",
		"-o", AddExeIfWindows("itch"),
		"-tags=alwaysdraw,minedev,screenshot",
		"-gcflags=all=-e -l -N",
		"itch.go",
	)

	if settings["no-vcs"] {
		cmd.Args = append(cmd.Args, "-buildvcs=false")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	misc.InfoLogger.Printf("%s", cmd.String())

	fmt.Printf("\n")
	err := cmd.Run()
	fmt.Printf("\n")

	if err != nil {
		return err, err.(*exec.ExitError).ExitCode()
	}

	return nil, 0
}

func BuildTsc(dir string) (error, int) {
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

	cmd.Dir = dir

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

	cmd.Dir = dir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	misc.InfoLogger.Printf("%s", cmd.String())

	fmt.Printf("\n")
	err = cmd.Run()
	fmt.Printf("\n")

	if err != nil {
		return err, err.(*exec.ExitError).ExitCode()
	}

	return nil, 0
}

func BuildSoundTsc() (error, int) {
	if err, exitCode := BuildTsc("./sound"); err != nil {
		return err, exitCode
	}

	// copy file
	const copySrc = "./sound/sound.js"
	const copyDst = "./web_build/sound.js"

	misc.InfoLogger.Printf("copying %s to %s", copySrc, copyDst)

	err := misc.CopyFile(
		copySrc, copyDst, 0664,
	)
	if err != nil {
		return err, 1
	}

	return nil, 0
}

func BuildWebPageTsc() (error, int) {
	if err, exitCode := BuildTsc("./web_build/scripts"); err != nil {
		return err, exitCode
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

func GlobMultiple(patterns []string) ([]string, error) {
	var matches []string

	for _, pattern := range patterns {
		m, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m...)
	}

	return matches, nil
}

func ZipFiles(
	files []string,
	filesBase string,
	dst string,
) error {
	// clean paths
	for i := range files {
		files[i] = strings.ReplaceAll(files[i], "\\", "/")
	}
	filesBase = strings.ReplaceAll(filesBase, "\\", "/")
	dst = strings.ReplaceAll(dst, "\\", "/")

	misc.InfoLogger.Printf("zipping")
	fmt.Printf("\n")
	for _, file := range files {
		fmt.Printf("  %s\n", file)
	}
	fmt.Printf("\n")
	fmt.Printf("to %s\n", dst)

	zipBuffer := new(bytes.Buffer)
	writer := zip.NewWriter(zipBuffer)

	for _, file := range files {
		rel, err := filepath.Rel(filesBase, file)
		if err != nil {
			return err
		}

		fileContent, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		fileWriter, err := writer.Create(rel)
		if err != nil {
			return err
		}

		_, err = io.Copy(fileWriter, bytes.NewReader(fileContent))
		if err != nil {
			return err
		}
	}

	if err := writer.Close(); err != nil {
		return err
	}

	if err := os.WriteFile(dst, zipBuffer.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func AddExeIfWindows(path string) string {
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	return path
}

// Get version string using git.
// Version string format is :
//
//	branchName-tag-commitCount-hash
//
// For example:
//
//	main--148-c9b1d68
//
// If dirty:
//
//	main--148-c9b1d68-dirty
//
// If release:
//
//	main--148-c9b1d68-release
func GetGitVersionString(isRelease bool) (string, error, int) {
	execCommand := func(name string, arg ...string) (string, error, int) {
		cmd := exec.Command(name, arg...)
		cmd.Stderr = os.Stderr

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", err, 1
		}

		if err = cmd.Start(); err != nil {
			return "", err, 1
		}

		stdoutCollected, err := io.ReadAll(stdout)
		if err != nil {
			return "", err, 1
		}

		if err = cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return "", exitErr, exitErr.ExitCode()
			} else {
				return "", err, 1
			}
		}

		if !utf8.Valid(stdoutCollected) {
			return "", fmt.Errorf("output from %s is not a valid utf8", cmd.String()), 1
		}

		return strings.TrimSpace(string(stdoutCollected)), nil, 0
	}

	var combined string

	// get current branch name
	out, err, exitCode := execCommand(
		"git", "rev-parse", "--abbrev-ref", "HEAD",
	)
	if err != nil {
		return "", err, exitCode
	}
	combined += out + "-"

	// get current tag
	out, err, exitCode = execCommand(
		"git", "describe", "--tags", "--abbrev=0",
	)
	if err != nil {
		return "", err, exitCode
	}
	combined += out + "-"

	// get commit count
	out, err, exitCode = execCommand(
		"git", "rev-list", "--count", "HEAD",
	)
	if err != nil {
		return "", err, exitCode
	}
	combined += out + "-"

	// get commit hash
	out, err, exitCode = execCommand(
		"git", "rev-parse", "--short", "HEAD",
	)
	if err != nil {
		return "", err, exitCode
	}
	combined += out

	// check if dirty
	out, err, exitCode = execCommand(
		"git", "status", "--porcelain",
	)
	if err != nil {
		return "", err, exitCode
	}

	if out != "" {
		combined += "-dirty"
	}

	// add release
	if isRelease {
		combined += "-release"
	}

	return combined, nil, 0
}
