package session

import "time"

// containerSession holds parsed names and paths for one container run.
type containerSession struct {
	proxyPort      uint16
	name           ContainerName
	workspace      Workspace
	project        ProjectName
	image          ImageName
	logDirectory   SessionLogDirectory
	logName        SessionLogName
	harnessStorage HarnessVolume
}

func newContainerSession(project projectIdentity, now time.Time) (containerSession, error) {
	logs, err := project.workspace.SessionLogs()
	if err != nil {
		return containerSession{}, err
	}
	logName, err := ParseSessionLogName(now.Format(sessionLogLayout))
	if err != nil {
		return containerSession{}, err
	}
	storage, err := resolveHarnessStorage(NoChoiceMade, project)
	if err != nil {
		return containerSession{}, err
	}
	return containerSession{
		name:      ContainerName{value: project.harness.prefix + "-" + now.Format("20060102-150405")},
		workspace: project.workspace, project: project.name, image: project.image,
		logDirectory: logs, logName: logName, harnessStorage: storage,
	}, nil
}
