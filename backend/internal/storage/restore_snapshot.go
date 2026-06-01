package storage

func RestoreSnapshot(id string, outputFile string) error {

	snapshot, err := GetSnapshot(id)

	if err != nil {
		return err
	}

	return RestoreFile(snapshot.Hash, outputFile)
}