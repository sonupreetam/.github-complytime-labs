# .github

Organization-wide community health files and profile for [complytime-labs](https://github.com/complytime-labs).

## What's here

- `profile/README.md` — Organization profile displayed on the [complytime-labs](https://github.com/complytime-labs) GitHub page

## Peribolos

This repository will apply peribolos to manage organization complytime-labs.

To use Peribolos to manage organization, the base requirement is Go setup.
When running Peribolos, it needs permission to access the organization,
and repository resources. The Github app installation access token has no
permission for endpoint user when peribolos updates members of organization. 
The Github app user access token could help. A user access token only has
permissions that both the user and the app have. For more details, please see
[Generating a user access token for a GitHub App](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app).

### Prerequisites
- [Registered a github app](https://docs.github.com/en/apps/creating-github-apps/registering-a-github-app/registering-a-github-app) and record the client ID
- [Installed the app](https://docs.github.com/en/apps/using-github-apps/installing-your-own-github-app) with reasonable permission
- [Enable the 'device flow'](https://docs.github.com/en/apps/creating-github-apps/writing-code-for-a-github-app/building-a-cli-with-a-github-app#about-device-flow-and-user-access-tokens) of the app
- [Generate a github app user access token](https://docs.github.com/en/apps/creating-github-apps/writing-code-for-a-github-app/building-a-cli-with-a-github-app#write-the-cli)

### References
- [Peribolos CLI](https://docs.prow.k8s.io/docs/components/cli-tools/peribolos/)
- [Peribolos source](https://github.com/kubernetes-sigs/prow/tree/main/cmd/peribolos)
