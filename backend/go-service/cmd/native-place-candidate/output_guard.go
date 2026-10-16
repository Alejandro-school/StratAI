package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateOutput(root string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("native place capture output root is required")
	}
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return fmt.Errorf("resolve system temporary directory: %w", err)
	}
	tempRoot, err = filepath.EvalSymlinks(tempRoot)
	if err != nil {
		return fmt.Errorf("physically resolve system temporary directory: %w", err)
	}
	tempRoot, err = filepath.Abs(tempRoot)
	if err != nil {
		return fmt.Errorf("normalize physical system temporary directory: %w", err)
	}
	outputRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve output root: %w", err)
	}
	logicalTempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		return fmt.Errorf("normalize system temporary directory: %w", err)
	}
	logicalRelative, err := strictPathDescendant(logicalTempRoot, outputRoot)
	if err != nil {
		return fmt.Errorf("compare output with system temporary directory: %w", err)
	}
	if logicalRelative == "" {
		return errors.New("native place capture output must be a dedicated directory below the system temporary directory")
	}

	current := logicalTempRoot
	for _, component := range strings.Split(logicalRelative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return fmt.Errorf("inspect native place output path component: %w", statErr)
		}
		reparse, reparseErr := pathIsReparsePoint(current, info)
		if reparseErr != nil {
			return fmt.Errorf("inspect native place output path component attributes: %w", reparseErr)
		}
		if reparse || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("native place capture output path must not contain symlinks or reparse points")
		}
		if !info.IsDir() {
			return errors.New("native place capture output path components must be directories")
		}
	}

	physicalOutputRoot, err := filepath.EvalSymlinks(outputRoot)
	if err != nil {
		return fmt.Errorf("physically resolve output root: %w", err)
	}
	physicalOutputRoot, err = filepath.Abs(physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("normalize physical output root: %w", err)
	}
	physicalRelative, err := strictPathDescendant(tempRoot, physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("compare physical output with physical system temporary directory: %w", err)
	}
	if physicalRelative == "" {
		return errors.New("native place capture output must physically remain below the system temporary directory")
	}
	expectedPhysicalRoot := filepath.Join(tempRoot, logicalRelative)
	expectedInfo, err := os.Stat(expectedPhysicalRoot)
	if err != nil {
		return fmt.Errorf("inspect expected physical output root: %w", err)
	}
	physicalInfo, err := os.Stat(physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("inspect physical output root: %w", err)
	}
	if !os.SameFile(expectedInfo, physicalInfo) {
		return errors.New("native place capture output physical resolution differs from its temporary path")
	}
	entries, err := os.ReadDir(physicalOutputRoot)
	if err != nil {
		return fmt.Errorf("inspect isolated physical output root: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("native place capture output root must be empty")
	}
	return nil
}

func strictPathDescendant(root, target string) (string, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relative == "." || relative == ".." || filepath.IsAbs(relative) ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", nil
	}
	return relative, nil
}
