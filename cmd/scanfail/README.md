# scanfail

scanfail finds unconfigured CodeQL languages for repos in a GitHub organization.

*Disclaimer: this tool only uses the GitHub API. Repos configured with `codeql.yaml` is not parsed and might lead to incorrect results.*


### Usage
Assuming your GitHub App private key is in `private-key.pem` and App ID is in `app-id.txt`. 
`reposfile.txt` contains newline delimited list of repo names.

```shell
scanfail -private-key private-key.pem -app-id app-id.txt -org-name myorg -repos reposfile.txt
```

After running, it will give stats output.
It wil create `results` directory in your current working directory if it does not already exist. It saves these following files in the
`results` directory:
`reposfile.json`,
`reposfile-misconfigured.json`,
`failedrepostories.txt`. 
If you run the program with the same repos file it wil overwrite the files in the `results` directory