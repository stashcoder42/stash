package config

import (
	"fmt"
	"io"
	"strconv"
)

type BlobsStorageType string

const (
	// Database
	BlobStorageTypeDatabase BlobsStorageType = "DATABASE"
	// Filesystem
	BlobStorageTypeFilesystem BlobsStorageType = "FILESYSTEM"
)

var AllBlobStorageType = []BlobsStorageType{
	BlobStorageTypeDatabase,
	BlobStorageTypeFilesystem,
}

func (e BlobsStorageType) IsValid() bool {
	switch e {
	case BlobStorageTypeDatabase, BlobStorageTypeFilesystem:
		return true
	}
	return false
}

func (e BlobsStorageType) String() string {
	return string(e)
}

func (e *BlobsStorageType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = BlobsStorageType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid BlobStorageType", str)
	}
	return nil
}

func (e BlobsStorageType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// WatcherScanMode defines the behavior when file changes are detected.
type WatcherScanMode string

const (
	// WatcherScanModeDisabled means file changes are ignored - no scan or identify triggered.
	WatcherScanModeDisabled WatcherScanMode = "DISABLED"
	// WatcherScanModeScan means scan files when changes are detected.
	WatcherScanModeScan WatcherScanMode = "SCAN"
	// WatcherScanModeScanAndIdentify means scan and identify files when changes are detected.
	WatcherScanModeScanAndIdentify WatcherScanMode = "SCAN_AND_IDENTIFY"
)

var AllWatcherScanMode = []WatcherScanMode{
	WatcherScanModeDisabled,
	WatcherScanModeScan,
	WatcherScanModeScanAndIdentify,
}

func (e WatcherScanMode) IsValid() bool {
	switch e {
	case WatcherScanModeDisabled, WatcherScanModeScan, WatcherScanModeScanAndIdentify:
		return true
	}
	return false
}

func (e WatcherScanMode) String() string {
	return string(e)
}

func (e *WatcherScanMode) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}
	*e = WatcherScanMode(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid WatcherScanMode", str)
	}
	return nil
}

func (e WatcherScanMode) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// ShouldScan returns true if the mode triggers scanning.
func (e WatcherScanMode) ShouldScan() bool {
	return e == WatcherScanModeScan || e == WatcherScanModeScanAndIdentify
}

// ShouldIdentify returns true if the mode triggers identification.
func (e WatcherScanMode) ShouldIdentify() bool {
	return e == WatcherScanModeScanAndIdentify
}
