package psa

type PatchOp struct {
	Op    Op          `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Op string
