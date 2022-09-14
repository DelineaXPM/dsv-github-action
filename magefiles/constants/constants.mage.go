package constants

// Since we are dealing with builds, having a constants file until using a config input makes it easy.

const (
	// ArtifactDirectory is a directory containing artifacts for the project and shouldn't be committed to source.
	ArtifactDirectory = ".artifacts"

	// PermissionUserReadWriteExecute is the permissions for the artifact directory.
	PermissionUserReadWriteExecute = 0o0700

	// CacheDirectory is where the cache for the project is placed, ie artifacts that don't need to be rebuilt often.
	CacheDirectory = ".cache"
)

const (

	// SecretFile is a local env file for testing integration with github action and not added to source control.
	SecretFile = ".cache/.secrets"
)

// Docker constants.
const (
	// LocalDockerRegistryPathQualified is the fully qualified path for the local built image to test against without publishing.
	LocalDockerRegistryPathQualified = "dsv-github-action"
	// DockerRegistryPathQualified is the fully qualified path for the github image this project will push to.
	DockerRegistryPathQualified = "registry.hub.docker.com/delineaxpm/dsv-github-action"
	// DockerTag is the tag to pull for integration tests, in this case, defaulting to latest.
	DockerTag = "latest"
)
