package models

// Native resource references are name digests, never asset-content digests.
type EconomyManifestEntry struct {
	Ordinal             int    `json:"ordinal"`
	DirectoryIndex      int    `json:"directory_index"`
	TypeIndex           int    `json:"type_index"`
	ReferenceNameSHA256 string `json:"reference_name_sha256"`
}

type EconomyManifestDecoded struct {
	Compression         string                 `json:"compression"`
	EncodedBytes        int                    `json:"encoded_bytes"`
	DecodedBytes        int                    `json:"decoded_bytes"`
	DecodedSHA256       string                 `json:"decoded_sha256"`
	OuterConsumedBits   int                    `json:"outer_consumed_bits"`
	OuterPaddingBits    int                    `json:"outer_padding_bits"`
	InnerConsumedBits   int                    `json:"inner_consumed_bits"`
	InnerPaddingBits    int                    `json:"inner_padding_bits"`
	TypeNameHashes      []string               `json:"type_name_hashes"`
	DirectoryNameHashes []string               `json:"directory_name_hashes"`
	Entries             []EconomyManifestEntry `json:"entries"`
}

type EconomyManifestSession struct {
	Sequence   int                     `json:"sequence"`
	Tick       int                     `json:"tick"`
	SourcePath string                  `json:"source_path"`
	Source     EconomyOpaqueField      `json:"source"`
	Status     string                  `json:"status"`
	Decoded    *EconomyManifestDecoded `json:"decoded"`
}
