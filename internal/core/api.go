package core

// DefaultMaxAttachmentSize is the default per-attachment size limit (50 MiB)
// applied by the /send API and `cc-connect send` when max_attachment_size_mb
// is unset. Exported so cmd/cc-connect can resolve the same default.
const DefaultMaxAttachmentSize int64 = 50 << 20
