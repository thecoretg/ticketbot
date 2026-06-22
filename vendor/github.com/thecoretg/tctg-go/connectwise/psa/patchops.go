package psa

type PatchOp struct {
	Op    Op     `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

type Op string
