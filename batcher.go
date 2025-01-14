//go:build ignore

// ====================================================
// program that converts audios in directory
// to .ogg files
//
// requires ffmpeg
//
// usage :
// 	go run batcher.go -s "./directory/to/audio/files/" -d "./directory/to/store/ogg/files/"
// ====================================================

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"minesweeper/misc"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	SrcDir string
	DstDir string
)

func init() {
	flag.Usage = func() {
		out := flag.CommandLine.Output()
		scriptName := misc.GetScriptName()

		fmt.Fprintf(out, "Usage of %s:\n", scriptName)
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "go run %s -s ./audio/input/directory/ -d ./output/directory/\n", scriptName)
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "requires ffmpeg\n")
		fmt.Fprintf(out, "\n")
		flag.PrintDefaults()
	}

	flag.StringVar(&SrcDir, "s", "", "source folder")
	flag.StringVar(&DstDir, "d", "", "destination folder")
}

func main() {
	if !misc.CheckExeExists("ffmpeg") {
		misc.ErrLogger.Fatal("couldn't find ffmpeg")
	}

	flag.Parse()

	// =========================
	// check if user gave path
	// =========================
	if len(SrcDir) <= 0 {
		misc.ErrLogger.Print("no source folder provided")
		flag.Usage()
		os.Exit(1)
	}
	if len(DstDir) <= 0 {
		misc.ErrLogger.Print("no destination folder provided")
		flag.Usage()
		os.Exit(1)
	}

	// =============================
	// check if SrcDir is folder
	// =============================
	if isDir, err := misc.IsDir(SrcDir); err != nil {
		misc.ErrLogger.Fatalf("failed to check if \"%s\" a folder: %v", SrcDir, err)
	} else if !isDir {
		misc.ErrLogger.Fatalf("\"%s\" is not a folder", SrcDir)
	}

	// =============================
	// collect files and folders
	// =============================
	// TODO: skip folders that has no audios
	var dirents []string
	var audioFiles []string
	{
		skippedRoot := false

		filepath.Walk(SrcDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				misc.ErrLogger.Printf("failed to walk \"%s\": %v", path, err)
				return filepath.SkipDir
			}

			if !skippedRoot {
				if isSame, err := misc.IsSamePath(path, SrcDir); err != nil {
					misc.ErrLogger.Fatalf("failed to check if path is source folder: %v", err)
				} else if isSame {
					skippedRoot = true
					return nil
				}
			}

			audioFileExts := []string{
				".mp3",
				".ogg",
				".wav",
				".flac",
			}

			if info.IsDir() {
				dirents = append(dirents, path)
			} else if info.Mode().IsRegular() {
				low := strings.ToLower(path)
				hasExt := false

				for _, ext := range audioFileExts {
					if strings.HasSuffix(low, ext) {
						hasExt = true
						break
					}
				}

				if hasExt {
					audioFiles = append(audioFiles, path)
				}
			}

			return nil
		})
	}

	fmt.Printf("\n")
	fmt.Printf("Audio Files\n")
	fmt.Printf("\n")
	for _, file := range audioFiles {
		fmt.Printf("    %s\n", file)
	}
	fmt.Printf("\n")

	var dstDirents []string = make([]string, len(dirents))
	var dstAudioFiles []string = make([]string, len(audioFiles))

	// ======================================
	// make everything relative to DstDir
	// ======================================
	for i, dirent := range dirents {
		if rel, err := filepath.Rel(SrcDir, dirent); err != nil {
			misc.ErrLogger.Fatalf("failed to make path local: %v", err)
		} else {
			dstDirents[i] = filepath.Join(DstDir, rel)
		}
	}
	for i, file := range audioFiles {
		if rel, err := filepath.Rel(SrcDir, file); err != nil {
			misc.ErrLogger.Fatalf("failed to make path local: %v", err)
		} else {
			dstAudioFiles[i] = filepath.Join(DstDir, rel)
		}
	}

	// ======================================
	// create folder structure
	// ======================================
	if err := misc.MkDir(DstDir); err != nil {
		misc.ErrLogger.Fatalf("failed to create directory: %v", err)
	}
	for _, dirent := range dstDirents {
		if err := misc.MkDir(dirent); err != nil {
			misc.ErrLogger.Fatalf("failed to create directory: %v", err)
		}
	}

	// ======================================
	// convert audio
	// ======================================
	var cmds []*exec.Cmd
	var stdouts []*bytes.Buffer
	var stderrs []*bytes.Buffer

	for i, file := range audioFiles {
		dstFile := dstAudioFiles[i]
		if before, found := strings.CutSuffix(dstFile, filepath.Ext(dstFile)); found {
			dstFile = before + ".ogg"
		}

		cmd := exec.Command(
			"ffmpeg",
			"-i", file, // input file
			"-vn",      // no video
			"-ac", "2", // 2 audio channel
			"-q:a", "5", // variable bitrate quality 5
			dstFile,
		)

		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		misc.InfoLogger.Print(cmd.String())

		if err := cmd.Start(); err != nil {
			misc.ErrLogger.Printf("failed to convert \"%s\": %v", file, err)
		} else {
			cmds = append(cmds, cmd)
			stdouts = append(stdouts, stdout)
			stderrs = append(stderrs, stderr)
		}
	}

	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			misc.ErrLogger.Printf("%s\n\n", cmd.String())
			os.Stdout.Write(stderrs[i].Bytes())
		}
	}
}
