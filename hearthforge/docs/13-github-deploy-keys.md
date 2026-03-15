# GitHub Deploy Keys

This guide explains why and how to use deploy keys with HearthForge and GitHub.

## Why Deploy Keys?

When a repo is private, HearthForge needs a way to `git clone` it into each developer's container during `forge hearthforge add-dev`. A GitHub Deploy key is a repo-scoped SSH key that lets HearthForge clone the repo read-only, without storing a personal token or GitHub password on the server.

Deploy keys are used **only for the initial bootstrap clone**. HearthForge never uses them for `git push`. After bootstrap, each developer uses their own GitHub SSH key and permissions to push commits from inside the container.

Deploy keys are now stored in `forge secrets` rather than plaintext files on disk. Use `forge hearthforge migrate-secrets` to migrate any existing `_deploy_keys/` directories.

## 1. Admin: Configure a Deploy Key for a Repo

Follow these steps once per project.

```bash
forge hearthforge add-project
```

When prompted:
- Enter the project id, repo URL, branch, stack, preview ports, and resources
- You can provide a GitHub HTTPS URL — HearthForge converts it to SSH internally when using the deploy key
- When asked "Generate a GitHub deploy key for this project?", answer `y`

After running, HearthForge will:
- Generate a deploy keypair and store it in `forge secrets` under `hearthforge.deploykeys.<project>`
- Print the **public key** and target repo URL

In GitHub, as the repo owner:
1. Open the repository
2. Go to **Settings → Deploy keys**
3. Click **"Add deploy key"**
4. Title: something like `forge-<project>`
5. Key: paste the public key printed by HearthForge
6. **Allow write access: leave unchecked** (read-only)
7. Save

If an earlier clone failed because the key was not yet added, run a manual clone inside the container:

```bash
ssh <dev>-<project>
cd /workspace/<project>
git clone git@github.com:owner/repo.git .
```

The deploy key is mounted at `/home/dev/.ssh/forge_deploy/id_ed25519` inside the container.

## 2. Developer: Connect the Container to Your GitHub Account

Developers push with their **own** GitHub identity from inside the dev container.

In the dev container (as user `dev`):

```bash
# Generate an SSH key
ssh-keygen -t ed25519 -C "your-email@example.com"
# Accept default path: /home/dev/.ssh/id_ed25519

# Show the public key
cat ~/.ssh/id_ed25519.pub
```

Add this key to your GitHub account:
1. Go to GitHub → your avatar → **Settings**
2. Choose **SSH and GPG keys**
3. Click **"New SSH key"**, give it a name (e.g. `forge-dev-container`), paste the key

Test SSH access:
```bash
ssh -T git@github.com
# Should show: Hi <username>! You've successfully authenticated...
```

Configure Git identity:
```bash
git config --global user.name "Your Name"
git config --global user.email "your-email@example.com"
```

Work as usual:
```bash
cd /workspace/<project>
git status
git commit -m "Your change"
git push
```

## 3. Responsibility Summary

| Actor | Responsibility |
|---|---|
| HearthForge | Creates per-project deploy keys, stores in `forge secrets`, mounts read-only into containers for bootstrap clone only |
| Admin | Adds deploy key public values as read-only Deploy keys on GitHub, grants developers write access to the repo |
| Developer | Configures own SSH key inside container, adds to GitHub account, pushes commits with own identity |
