package cmd

import (
	"benetnasch/app/infra/ai/evaldata"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const maxVisionScoreFileBytes int64 = 4 * 1024 * 1024

var visionEvalCmd = &cobra.Command{
	Use:   "vision-eval",
	Short: "validate offline Vision evaluation scores",
	Args:  cobra.NoArgs,
}

var visionEvalReportCmd = &cobra.Command{
	Use:   "report",
	Short: "aggregate a completed Vision score file without side effects",
	Args:  cobra.NoArgs,
	RunE:  runVisionEvalReport,
}

func init() {
	visionEvalReportCmd.Flags().String("scores", "", "path to a JSON array or JSONL file of human Vision scores")
	if err := visionEvalReportCmd.MarkFlagRequired("scores"); err != nil {
		panic(err)
	}
	visionEvalCmd.AddCommand(visionEvalReportCmd)
	rootCmd.AddCommand(visionEvalCmd)
}

func runVisionEvalReport(cmd *cobra.Command, _ []string) error {
	path, err := cmd.Flags().GetString("scores")
	if err != nil {
		return err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("scores path is required")
	}
	file, err := openExternalVisionScoreFile(path)
	if err != nil {
		return fmt.Errorf("open Vision score file: %w", err)
	}
	defer file.Close()

	scores, err := decodeVisionScores(io.LimitReader(file, maxVisionScoreFileBytes+1))
	if err != nil {
		return fmt.Errorf("decode Vision score file: %w", err)
	}
	cases, err := evaldata.LoadVisionCases()
	if err != nil {
		return fmt.Errorf("load Vision evaluation cases: %w", err)
	}
	report, err := evaldata.AggregateVisionScores(cases, scores)
	if err != nil {
		return fmt.Errorf("aggregate Vision scores: %w", err)
	}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("encode Vision evaluation report: %w", err)
	}
	if !report.Pass {
		return errors.New("Vision evaluation did not pass the configured quality gate")
	}
	return nil
}

func openExternalVisionScoreFile(path string) (*os.File, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("scores path is required")
	}
	fullPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("resolve score path: %w", err)
	}
	repoRoot, err := findVisionEvaluationRepositoryRoot()
	if err != nil {
		return nil, err
	}
	resolvedPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return nil, fmt.Errorf("resolve score file links: %w", err)
	}
	resolvedPath, err = filepath.Abs(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("resolve linked score path: %w", err)
	}
	if !visionEvaluationPathOutsideRepository(repoRoot, fullPath) ||
		!visionEvaluationPathOutsideRepository(repoRoot, resolvedPath) {
		return nil, errors.New("scores path must be outside the repository")
	}
	if !sameVisionEvaluationPath(fullPath, resolvedPath) {
		return nil, errors.New("scores path must not be a symbolic link or reparse point")
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("stat score file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("scores path must be a regular file")
	}
	return os.Open(resolvedPath)
}

func findVisionEvaluationRepositoryRoot() (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	current, err := filepath.Abs(workingDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(current, "go.mod")); statErr == nil && !info.IsDir() {
			return filepath.Clean(current), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("repository root could not be located")
		}
		current = parent
	}
}

func visionEvaluationPathOutsideRepository(repoRoot, candidate string) bool {
	if !strings.EqualFold(filepath.VolumeName(repoRoot), filepath.VolumeName(candidate)) {
		return true
	}
	relative, err := filepath.Rel(repoRoot, candidate)
	if err != nil || relative == "." || relative == ".." || filepath.IsAbs(relative) {
		return false
	}
	parentPrefix := ".." + string(os.PathSeparator)
	return strings.HasPrefix(relative, parentPrefix)
}

func sameVisionEvaluationPath(left, right string) bool {
	if filepath.VolumeName(left) != "" || filepath.VolumeName(right) != "" {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func decodeVisionScores(reader io.Reader) ([]evaldata.HumanVisionScore, error) {
	if reader == nil {
		return nil, errors.New("score reader is nil")
	}
	contents, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(contents)) > maxVisionScoreFileBytes {
		return nil, fmt.Errorf("score file exceeds %d bytes", maxVisionScoreFileBytes)
	}
	trimmed := bytes.TrimSpace(contents)
	if len(trimmed) == 0 {
		return nil, errors.New("score file is empty")
	}
	if trimmed[0] == '[' {
		var scores []evaldata.HumanVisionScore
		if err := decodeStrictJSON(trimmed, &scores); err != nil {
			return nil, fmt.Errorf("score JSON array is invalid: %w", err)
		}
		if len(scores) == 0 {
			return nil, errors.New("score file is empty")
		}
		return scores, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(trimmed))
	scanner.Buffer(make([]byte, 1024), 256*1024)
	scores := make([]evaldata.HumanVisionScore, 0)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var score evaldata.HumanVisionScore
		if err := decodeStrictJSON(line, &score); err != nil {
			return nil, fmt.Errorf("score JSONL line %d is invalid: %w", len(scores)+1, err)
		}
		scores = append(scores, score)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan score JSONL: %w", err)
	}
	if len(scores) == 0 {
		return nil, errors.New("score file is empty")
	}
	return scores, nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}
