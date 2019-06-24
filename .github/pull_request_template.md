<!-- If you're making a doc PR or something tiny where the below is irrelevant, just delete this
template and use a short description. -->

<details>
  <summary>PR Checklist</summary>
  
### PR Structure

* [ ] This PR has reasonably narrow scope (if not, break it down into smaller PRs).
* [ ] This PR avoids mixing refactoring changes with feature changes (split into two PRs
  otherwise).
* [ ] This PR's title starts with name of package that is most changed in the PR, ex.
  `services/friendbot`

### Thoroughness

* [ ] This PR adds tests for the most critical parts of the new functionality or fixes.
* [ ] I've updated any docs ([developer docs](https://www.stellar.org/developers/reference/), `.md`
  files, etc... affected by this change). Take a look in the `docs` folder for a given service,
  like [this one](https://github.com/stellar/go/tree/master/services/horizon/internal/docs).

### Release planning

* [ ] I've updated the relevant CHANGELOG ([here](services/horizon/CHANGELOG.md) for Horizon) if
  needed with deprecations, added features, breaking changes, and DB schema changes.
* [ ] I've decided if this PR requires a new major/minor version according to
  [semver](https://semver.org/), or if it's mainly a patch change. The PR is targeted at the next
  release branch if it's not a patch change.
</details>

### Summary

[TODO]

### Goal and scope

[TODO]

### Summary of changes

[TODO]

### Known limitations & issues

[TODO]

### What shouldn't be reviewed

[TODO]
