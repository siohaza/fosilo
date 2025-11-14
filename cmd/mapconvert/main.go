package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/siohaza/fosilo/internal/mapmeta"
	"github.com/spf13/cobra"
)

var (
	inputDir  string
	outputDir string
)

var rootCmd = &cobra.Command{
	Use:   "mapconvert [files...]",
	Short: "convert aos map metadata from pyspades to toml format",
	Run:   runConvert,
}

func init() {
	rootCmd.Flags().StringVarP(&inputDir, "input", "i", "file1.txt", "Input file/directory with Pyspades metadata files")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "maps", "Output directory for TOML files")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runConvert(cmd *cobra.Command, args []string) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	files, err := getInputFiles(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get input files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No input files found")
		os.Exit(1)
	}

	converted := 0
	skipped := 0
	failed := 0

	for _, file := range files {
		metadata, err := convertMetadataFile(file)
		if err != nil {
			fmt.Printf("SKIP %s: %v\n", filepath.Base(file), err)
			skipped++
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(file), ".txt")
		outputPath := filepath.Join(outputDir, baseName+".toml")

		if err := writeToml(outputPath, metadata); err != nil {
			fmt.Printf("FAIL %s: %v\n", baseName, err)
			failed++
			continue
		}

		fmt.Printf("OK   %s -> %s\n", filepath.Base(file), filepath.Base(outputPath))
		converted++
	}

	fmt.Printf("\nSummary: %d converted, %d skipped, %d failed\n", converted, skipped, failed)
}

func getInputFiles(args []string) ([]string, error) {
	var files []string

	if len(args) > 0 {
		for _, arg := range args {
			info, err := os.Stat(arg)
			if err != nil {
				return nil, fmt.Errorf("cannot access %s: %w", arg, err)
			}

			if info.IsDir() {
				dirFiles, err := filepath.Glob(filepath.Join(arg, "*.txt"))
				if err != nil {
					return nil, fmt.Errorf("failed to list files in %s: %w", arg, err)
				}
				files = append(files, dirFiles...)
			} else {
				files = append(files, arg)
			}
		}
	} else {
		dirFiles, err := filepath.Glob(filepath.Join(inputDir, "*.txt"))
		if err != nil {
			return nil, fmt.Errorf("failed to list files in %s: %w", inputDir, err)
		}
		files = append(files, dirFiles...)
	}

	return files, nil
}

func convertMetadataFile(filename string) (*mapmeta.Metadata, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	content := string(data)
	metadata, err := mapmeta.Parse(data)
	if err != nil {
		return nil, err
	}

	applySpawnFallbacks(metadata, content)
	applyEntityFallbacks(metadata, content)

	warnings := validateMetadata(metadata)
	if len(warnings) > 0 {
		baseName := filepath.Base(filename)
		for _, warning := range warnings {
			fmt.Printf("WARN %s: %s\n", baseName, warning)
		}
	}

	return metadata, nil
}

func applySpawnFallbacks(meta *mapmeta.Metadata, content string) {
	if len(meta.Spawns.Blue) == 0 {
		meta.Spawns.Blue = extractFunctionSpawns(content, "blue")
	}
	if len(meta.Spawns.Green) == 0 {
		meta.Spawns.Green = extractFunctionSpawns(content, "green")
	}
	if len(meta.Spawns.Blue) == 0 && len(meta.Spawns.BlueArea) == 0 {
		meta.Spawns.BlueArea = extractSpawnArea(content, "BLUE")
	}
	if len(meta.Spawns.Green) == 0 && len(meta.Spawns.GreenArea) == 0 {
		meta.Spawns.GreenArea = extractSpawnArea(content, "GREEN")
	}
}

func applyEntityFallbacks(meta *mapmeta.Metadata, content string) {
	blueFlagMissing := len(meta.Entities.Blue.Flag) == 0
	blueBaseMissing := len(meta.Entities.Blue.Base) == 0
	greenFlagMissing := len(meta.Entities.Green.Flag) == 0
	greenBaseMissing := len(meta.Entities.Green.Base) == 0

	if !blueFlagMissing && !blueBaseMissing && !greenFlagMissing && !greenBaseMissing {
		return
	}

	blueFlag, blueBase := extractEntityLocations(content, "BLUE")
	greenFlag, greenBase := extractEntityLocations(content, "GREEN")

	if blueFlagMissing && len(blueFlag) == 3 {
		meta.Entities.Blue.Flag = blueFlag
	}
	if blueBaseMissing && len(blueBase) == 3 {
		meta.Entities.Blue.Base = blueBase
	}
	if greenFlagMissing && len(greenFlag) == 3 {
		meta.Entities.Green.Flag = greenFlag
	}
	if greenBaseMissing && len(greenBase) == 3 {
		meta.Entities.Green.Base = greenBase
	}
}

func extractFunctionSpawns(content, team string) [][]float64 {
	teamCheck := "blue_team"
	if team == "green" {
		teamCheck = "green_team"
	}

	pattern := fmt.Sprintf(`if\s+connection\.team\s+is\s+connection\.protocol\.%s[\s\S]*?return\s*\(\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*\)`, teamCheck)
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(content); len(matches) == 4 {
		x, _ := strconv.ParseFloat(matches[1], 64)
		y, _ := strconv.ParseFloat(matches[2], 64)
		z, _ := strconv.ParseFloat(matches[3], 64)
		return [][]float64{{x, y, z}}
	}

	return [][]float64{}
}

func extractSpawnArea(content, team string) []float64 {
	rectName := team + "_RECT"

	pattern := fmt.Sprintf(`%s\s*=\s*\(\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*\)`, rectName)
	re := regexp.MustCompile(pattern)
	if matches := re.FindStringSubmatch(content); len(matches) == 5 {
		x1, _ := strconv.ParseFloat(matches[1], 64)
		y1, _ := strconv.ParseFloat(matches[2], 64)
		x2, _ := strconv.ParseFloat(matches[3], 64)
		y2, _ := strconv.ParseFloat(matches[4], 64)
		return []float64{x1, y1, x2, y2}
	}

	return []float64{}
}

func extractEntityLocations(content, team string) (flag, base []float64) {
	flagPattern := fmt.Sprintf(`if\s+entity_id\s+==\s+%s_FLAG[\s\S]*?return\s*\(\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*\)`, team)
	flagRe := regexp.MustCompile(flagPattern)
	if matches := flagRe.FindStringSubmatch(content); len(matches) == 4 {
		x, _ := strconv.ParseFloat(matches[1], 64)
		y, _ := strconv.ParseFloat(matches[2], 64)
		z, _ := strconv.ParseFloat(matches[3], 64)
		flag = []float64{x, y, z}
	}

	basePattern := fmt.Sprintf(`if\s+entity_id\s+==\s+%s_BASE[\s\S]*?return\s*\(\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*\)`, team)
	baseRe := regexp.MustCompile(basePattern)
	if matches := baseRe.FindStringSubmatch(content); len(matches) == 4 {
		x, _ := strconv.ParseFloat(matches[1], 64)
		y, _ := strconv.ParseFloat(matches[2], 64)
		z, _ := strconv.ParseFloat(matches[3], 64)
		base = []float64{x, y, z}
	}

	return
}

func validateMetadata(meta *mapmeta.Metadata) []string {
	var warnings []string

	if meta.Metadata.Name == "" {
		warnings = append(warnings, "missing required field 'name'")
	}

	hasBlueSpawn := len(meta.Spawns.Blue) > 0 || len(meta.Spawns.BlueArea) > 0
	hasGreenSpawn := len(meta.Spawns.Green) > 0 || len(meta.Spawns.GreenArea) > 0

	if !hasBlueSpawn {
		warnings = append(warnings, "no blue team spawn locations defined")
	}
	if !hasGreenSpawn {
		warnings = append(warnings, "no green team spawn locations defined")
	}

	if len(meta.Entities.Blue.Flag) == 0 {
		warnings = append(warnings, "missing blue flag position")
	}
	if len(meta.Entities.Green.Flag) == 0 {
		warnings = append(warnings, "missing green flag position")
	}
	if len(meta.Entities.Blue.Base) == 0 {
		warnings = append(warnings, "missing blue base position")
	}
	if len(meta.Entities.Green.Base) == 0 {
		warnings = append(warnings, "missing green base position")
	}

	for _, spawn := range meta.Spawns.Blue {
		if !isValidCoordinate(spawn) {
			warnings = append(warnings, "blue spawn has invalid coordinates")
			break
		}
	}
	for _, spawn := range meta.Spawns.Green {
		if !isValidCoordinate(spawn) {
			warnings = append(warnings, "green spawn has invalid coordinates")
			break
		}
	}

	if len(meta.Entities.Blue.Flag) > 0 && !isValidCoordinate(meta.Entities.Blue.Flag) {
		warnings = append(warnings, "blue flag has invalid coordinates")
	}
	if len(meta.Entities.Green.Flag) > 0 && !isValidCoordinate(meta.Entities.Green.Flag) {
		warnings = append(warnings, "green flag has invalid coordinates")
	}
	if len(meta.Entities.Blue.Base) > 0 && !isValidCoordinate(meta.Entities.Blue.Base) {
		warnings = append(warnings, "blue base has invalid coordinates")
	}
	if len(meta.Entities.Green.Base) > 0 && !isValidCoordinate(meta.Entities.Green.Base) {
		warnings = append(warnings, "green base has invalid coordinates")
	}

	return warnings
}

func isValidCoordinate(coord []float64) bool {
	if len(coord) != 3 {
		return false
	}
	for _, v := range coord {
		if v < 0 || v > 512 {
			return false
		}
	}
	return true
}

func writeToml(filename string, metadata *mapmeta.Metadata) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	encoder.Indent = ""
	return encoder.Encode(metadata)
}
