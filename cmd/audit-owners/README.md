# audit-owners

This utility reviews the OWNERS files on a project and attempts to detect those
that are not collaborators.

## Usage

A valid GitHub Token needs to be set as an environment variable named `GITHUB_TOKEN`.

*Flags:*

* `-r`: The root location of the repository to start looking
* `-b`: Creates a bulleted list that can be sent to the mailing list where
  people can become members of the Kubernetes org on GitHub
* `-a`: GitHub names found in OWNERS files who are not collaborators will be
  added as read only (pull) collaborators to the repo. Requires that the token
  being used to interact with the GitHub API as repo admin writes

```
$ audit-owners -r ~/Code/k8s/charts
GitHub Login "CaptTofu" found in OWNERS but not a collaborator
```