package sandbox

import _ "embed"

//go:embed cloud-init.yaml
var cloudInitTemplate []byte
