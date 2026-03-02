# How to release

This project uses **GitHub Releases** for binaries and **GitHub Actions** to build and attach assets when you publish a release. You create the release first; the workflows then build and attach artifacts.

## Flow (recommended)

1. **Create a new release** on GitHub:
   - Go to [Releases](https://github.com/alvarolobato/iptv-proxy/releases) → **Draft a new release**.
   - Choose a **tag** (e.g. `v1.0.0`). Create the tag if it doesn’t exist (e.g. from `master`).
   - Set the release title and description.
   - You can save as **Draft** or go straight to **Publish release**.

2. **Publish the release** (if you saved as draft, click **Publish release**).

3. **Workflows run automatically:**
   - **Release binaries** (`.github/workflows/release.yml`): builds the binary for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64, arm64), then **attaches** the archives to the release. After a few minutes, the release page will list e.g. `iptv-proxy_linux_amd64.tar.gz`, `iptv-proxy_darwin_arm64.tar.gz`, `iptv-proxy_windows_amd64.zip`, etc.
   - **Release Docker Image** (`.github/workflows/release-docker.yml`): builds and pushes the Docker image to Docker Hub (requires `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` secrets).

You do **not** need to run an action manually to “build and create” the release: **you create (and publish) the release; the workflows attach the binaries and push the image.**

## Summary

| Step | What you do | What happens |
|------|-------------|--------------|
| 1 | Create a release with a tag (e.g. `v1.0.0`) and publish it | Release appears on the repo |
| 2 | (Nothing) | **Release binaries** workflow runs and attaches Linux/macOS/Windows binaries to the release |
| 3 | (Nothing) | **Release Docker Image** workflow runs and pushes the image to Docker Hub (if secrets are set) |

## Requirements

- **Binaries:** No extra secrets. The workflow uses the default `GITHUB_TOKEN` to upload assets to the release.
- **Docker image:** Repository secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` must be set for the Docker image to be pushed.

## Creating the tag from the command line

If you prefer to create the tag and push it, then create the release from the tag:

```bash
git checkout master
git pull
git tag v1.0.0
git push origin v1.0.0
```

Then on GitHub: **Releases** → **Draft a new release** → select the existing tag `v1.0.0`, add title/notes, and **Publish release**. The workflows will run as above.

Note: Pushing a tag alone does **not** create a GitHub Release in this setup. The **Release binaries** workflow is triggered by **release: published**, not by tag push. So you must create (and publish) the release in the UI (or via API) for the binaries to be built and attached.
