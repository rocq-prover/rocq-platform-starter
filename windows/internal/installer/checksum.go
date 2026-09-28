package installer

import (
	sharedinstaller "github.com/rocq-prover/rocq-platform-starter/shared/installer"
)

// VerifySHA256 checks the SHA256 hash of the file at path.
// If expected is empty, the check is skipped.
var VerifySHA256 = sharedinstaller.VerifySHA256
