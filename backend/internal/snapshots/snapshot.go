package snapshots

type Snapshot struct {
	ID       string `json:"id"`
	FileName string `json:"filename"`
	Hash     string `json:"hash"`
	Version  int    `json:"version"`
}