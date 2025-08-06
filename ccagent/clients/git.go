package clients

import gitclient "ccagent/clients/git"

func NewGitClient() GitClient {
	return gitclient.NewGitClient()
}
