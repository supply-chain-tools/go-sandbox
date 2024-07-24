# repofetch + gitsearch tutorial

## Clone PyPI org

```sh
mkdir tutorial
cd tutorial
repofetch github.com/pypi
```
This will create the directory `pypi` and clone all the repositories of the [github.com/pypi](https://github.com/orgs/pypi/repositories) into that directory.
```
pypi/
|--camo/
|--cdn-log-archiver/
...
```

## Search

### Prerequisite
Set up a wrapper script with colors and pager ([../cmd/gitesarch/gs](../cmd/gitsearch/gs)). The examples will assume the
wrapper command is `gs`.

### Basic search
This performs a normalized search for `test` (case-insensitive and that potentially repeated delimiters `-`, `_` and `.`
are interpreted as a single delimiter):
```
$ gs test
```

#### How repositories to search are resolved
`gs` starts by looking to see if it's currently inside a Git repository by looking for a `.git` directory in the current
working directory, then each directory up to `/`. Afterwards it will check if it's currently in a directory with Git repos,
or if there is Git repos one directory below. In this case `pypi` is interpreted as an organization, with a list of
repos below. `repofetch` creates the expected org structure to search
multiple repos in multiple orgs.

#### Search output
```
$ gs test
pypi/cdn-log-archiver/.gitignore:
31:# Unit test / coverage reports
35:nosetests.xml
...
```
The first line is the path to the file, which is the org `pypi`, repo `cnd-log-archiver` then path inside the repo
`.gitignore`
Then the lines that match are displayed, with the matching word in a different color. There are a number of flags to
adjust how the output is displayed, which can be found in `gs -h`.

### Search all history/branches
#### Search all history
The all history search is a bit slow, so we'll go into a specific repo
```
$ cd pypi/infra
```

```
$ gs --all-history secret
terraform/file-hosting/vcl/files.vcl:
3:    declare local var.AWS-Secret-Access-Key STRING;
   refs/remotes/origin/ComputeAtEdge_Test[2023-04-12:86580a <- 2022-09-12:985133]
   refs/remotes/origin/ease_off_on_b2[2023-04-12:86580a <- 2022-09-12:985133]
   refs/remotes/origin/geoip[2023-04-12:86580a <- 2022-09-12:985133]
   refs/remotes/origin/main[2023-04-12:86580a <- 2022-09-12:985133]
  ^refs/remotes/origin/streaming_miss[2023-01-10:065a2a <- 2022-09-12:985133]
...
```
The output starts with the path to the file and the line match as before. The next few lines is the branches
that matched. The matched line existed on the branch `ComputeAtEdge_Test` from commit `985133` on `2022-09-12` to commit
`86580a` on `2023-04-12`. The leading `^` on the last line means that the line is still present at the tip of that
branch, meaning that the most recent commit on that branch is `065a2a` and contains the line. Any matching tags
will be listed below the branches.

#### Deduplication
The results are deduplicated as long as the matching line and path is the same. So if a line is moved inside
the file it will be treated as the same occurrence. If the line is changed but still matches it will be listed as a
separate result. The line number shown in the match above (3) is the line number when the occurrence was first seen.

#### Search all branches
```
$ gs --all-branches secret
terraform/file-hosting/vcl/files.vcl:
205:  declare local var.B2SecretKey STRING;
  ^refs/remotes/origin/ComputeAtEdge_Test[2023-06-30:bbafc4]
  ^refs/remotes/origin/ease_off_on_b2[2023-09-06:a1caf5]
  ^refs/remotes/origin/geoip[2023-06-24:0b1e24]
  ^refs/remotes/origin/main[2024-03-22:aeb7b5]
...
```

The output is similar to `all-history`, but now since only the most recent commit on each branch is searched, there 
is no range of commits, only that the match was present on the tip of the branch (marked with leading `^`) and the 
commit hash and date of the match.

### Search typos in domains
Go back to the directory that contains the `pypi` org directory:
```
cd ../..
```

Search for typosquatting variations of `pypi.org`. This will take typos of `pypi` and combine with TLDs that are not
`.org`
```
gs --type domain --match typo pypi.org
```

This will give substring results that we might not want, like `vnd.pypi.simple.v1`. To only match strings ending in
a valid TLD, use `--anchor-end`
```
gs --type domain --match typo --anchor-end pypi.org
```

This will still give subdomain matches like `test.pypi.org`. Use `--anchor-beginning` to not match subdomains
```
gs --type domain --match typo --anchor-beginning --anchor-end pypi.org
```

The results include `pypi.io`, `pypi.pid`. The idea is to enter all the domains you know about and then see what's 
left, so let's add`pypi.io` and `pypi.pid` to the list
```
gs --type domain --match typo --anchor-beginning --anchor-end pypi.org pypi.io pypi.pid
```

This leaves only `pypi.dev`, `pypi.search`. The results are only related to different TLDs, not typos of `pypi` in the
example. To see a list of all variations that was searched for use `--output-search-strings`
```
gs --type domain --match typo --anchor-beginning --anchor-end --output-search-strings pypi.org pypi.io pypi.pid
```
At the time of writing there is about 67k variations due to the combinatorial explosion between typos of `pypi` and
all the TLDs.

#### How typo variations are generated
A mix of strategies are used to generate variations (largely based on the [SpellBound paper](https://arxiv.org/abs/2003.03471)):
 - duplication of characters, e.g. `abc` -> `aabc`
 - omitting a character, e.g. `abc` -> `bc`
 - swap two characters, e.g. `abc` -> `bac`
 - reorder delimited words, e.g. `foo-bar` -> `bar-foo`
 - replace characters based on QWERTY layout, e.g. `q` is next to `w`
 - replace characters based on being similar looking, e.g. `1` and `l`