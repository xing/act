# XING `act` fork with patches applied

This repository is a fork of `nektos/act` with a few changes:
- The `master` branch is automatically updated with the latest changes from `nektos/act` [workflow](https://github.com/xing/act/actions/workflows/sync-upstream.yml)
- We have an orphan branch called `distribution`, which contains the patches and our workflows to sync the `master` branch and apply the patches
- We push the patched master branch as `patched-master`
- We publish a new release each time something changes (upstream `master` is updated or patches are updated)

## Status

[![Synced with upstream](https://github.com/xing/act/actions/workflows/sync-upstream.yml/badge.svg)](https://github.com/xing/act/actions/workflows/sync-upstream.yml)  
[![Build with patches](https://github.com/xing/act/actions/workflows/build-with-patches.yml/badge.svg)](https://github.com/xing/act/actions/workflows/build-with-patches.yml)  


## Development workflow

We have two different types of patches:
- "forever downstream" patches, so that we can deploy as `xing/act` and which will never be upstreamed
- "temporary" patches, which are PRs open on `nektos/act` that we expect to be merged soon

### Creating "forever downstream" patches

In order to patch something that will never get upstreamed, we need to branch off of `master` and push the changes as `patch/<patch-name>`.
We then create a draft PR against the `master` branch on `xing/act` to get the patch URL.

Next we branch off of `distribution` and add the URL of the previously created PR to the `patches` file and create a PR against the `distribution` branch on `xing/act`.
This PR will try to apply all patches and run the tests. Once it succeeds, we can merge it so that the `patches` file on `distribution` contains all required patches.

### Creating "temporary" patches

#### Creating a patch for our own PRs (trusted)

We branch off of `distribution` and add the URL of our PR to the `patches` file and create a PR against the `distribution` branch on `xing/act`.
This PR will try to apply all patches and run the tests. Once it succeeds, we can merge it so that the `patches` file on `distribution` contains all required patches.

In case merge conflicts arise, restart and follow the instructions for untrusted PRs.


#### Creating a patch for other PRs (untrusted)

> Important to note here: We need to thoroughly review the code that we merge in here, because it gets built automatically and is then being used in the act-app.

In order to create a patch for untrusted PRs we need to branch off of `patched-master` and squash merge the changes into our newly created branch.
All potential conflicts need to be resolved, so that the resulting patch can apply cleanly.

Next we need to push the changes to `xing/act` and create a patch using the compare URL with the start/end commit SHAs.
Example: `https://github.com/xing/act/compare/<patched-master-sha>...<our-branch-sha>.patch`

This patch URL can now be added to the `patches` file as described for trusted PRs.

Since we are "pinning" the PR changes to commit SHAs that exist in `xing/act` we need to repeat the above process if the upstream PR gets updated.
