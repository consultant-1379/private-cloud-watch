package version

type versionConfig struct {
	Version        string
	BuildDatetime  string
	CommitDatetime string
	CommitID       string
	ShortCommitID  string
	BranchName     string
	GitTreeIsClean bool
	IsOfficial     bool
}

func Version() versionConfig {
	gitTreeIsClean := (cruxGitTreeIsClean == "true")
	return versionConfig{
		Version:        gitTag(),
		BuildDatetime:  cruxBuildDatetime,
		CommitDatetime: cruxCommitDatetime,
		CommitID:       cruxCommitID,
		ShortCommitID:  cruxShortCommitID,
		BranchName:     cruxBranchName,
		GitTreeIsClean: gitTreeIsClean,
		IsOfficial:     isOfficial(),
	}
}
