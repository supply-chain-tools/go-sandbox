# gitsearch

A multi-string, multi-git-repo, all history, exact/fuzzy searcher.

Having a list of strings of interest is not uncommon, especially in security, and it's useful to search
for them in all the history of one or more git repos. Examples:
 - Domains/URLs
 - Package names
 - Secrets
 - Dorks
 - Brand/product names

This tool is meant to be used for exploring and to support building more targeted static analysis tools.
I.e. it's not meant to replace e.g. dedicated secret scanners or SASTs.

### Features
 - Search one or more local git repos, or collections of git repos (e.g. organizations)
 - Match multiple keywords in one pass
 - Match name confusion (including typosquatting) variations of package names and domains, including matching other TLDs
- `git grep`/`rg` style UX with colors and `less` pager, or plaintext pipe/redirect

### What It's Not
 - `grep`, `git grep` or `rg`; it does not support regular expressions!
 - Search engine; it does not index the data

### Design Goals
 - Be thorough: search everything in Git, i.e. all paths, all branches/tags/commits, and dangling objects. List all search terms that are being matched in a line, including if they overlap.
 - Be convenient: search multiple repos and strings at once. Search for exact, normalized, typos, or fuzzy matches. Built-in support for domains, to help search for typo variations with legitimate TLDs. Built in support for package names.
 - Be reasonably fast.

### Technical Details
 - Multi-string search is implemented as a Nondeterministic Finite Automata (NFA) using a [Trie](https://en.wikipedia.org/wiki/Trie). 
It might be optimized in the future.
 - It does not use [Aho-Corasick](https://en.wikipedia.org/wiki/Aho%E2%80%93Corasick_algorithm) or any SIMD skipping like [ripgrep](https://github.com/BurntSushi/ripgrep) does with [Teddy](https://github.com/rust-lang/regex/blob/1.10.4/regex-automata/src/util/prefilter/teddy.rs).
 - There is no optimization of single string search using e.g. [Boyer-Moore](https://en.wikipedia.org/wiki/Boyer%E2%80%93Moore_string-search_algorithm) or skipping with [memchr](https://man7.org/linux/man-pages/man3/memchr.3.html).
 - Typosquatting is implemented by generating typo variations and adding them to the Trie. The typo variations can be outputted so it can be used with other tools. 
 - No optimization for large files: they are read entirely into memory, and multiple are processed concurrently. Given the focus on Git repos, where files should be small, and even enforced in some cases: [GitHub has a hard limit of 100MB](https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github) and [Gitlab has a soft limit of 100MB](https://docs.gitlab.com/ee/user/free_push_limit.html). [LFS](https://git-lfs.com/) is not a focus.
 - It does not currently support symlinks.
 - It does not currently support submodules.
 - it does not currently support `.gitingore` and other "smart" filtering.
 - There is partial UTF-8 support, but at a performance cost.
 - Fuzzy matching using [Trigrams](https://en.wikipedia.org/wiki/Trigram) would be an interesting feature.