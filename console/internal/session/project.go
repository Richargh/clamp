package session

import "strconv"

// projectIdentity holds stable values derived once from a resolved workspace
// and a parsed harness. Session timestamps are not part of project identity.
type projectIdentity struct {
	harness   Harness
	workspace Workspace
	name      ProjectName
	key       uint32
	toolImage ImageName
	image     ImageName
}

func newProjectIdentity(harness Harness, workspace Workspace) projectIdentity {
	key := projectChecksum([]byte(workspace.String()))
	tool := "clamp-" + harness.name
	return projectIdentity{
		harness:   harness,
		workspace: workspace,
		name:      ProjectNameFor(workspace),
		key:       key,
		toolImage: ImageName{value: tool},
		image:     ImageName{value: tool + "-" + strconv.FormatUint(uint64(key), 10)},
	}
}

func (p projectIdentity) credentialVolume(policy AuthPolicy) VolumeName {
	name := p.harness.prefix
	if policy.IsProject() {
		name += "-" + p.name.String() + "-" + strconv.FormatUint(uint64(p.key), 10)
	}
	return VolumeName{value: name}
}
