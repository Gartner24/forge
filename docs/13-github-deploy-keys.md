# GitHub deploy keys and developer access

This guide explains **why** and **how** to use deploy keys with Forge and GitHub.

Why deploy keys?
- When a repo is **private**, Forge still needs a way to `git clone` it into each developer’s container during `devctl add-dev`.
- A GitHub **Deploy key** is a repo-scoped SSH key that lets Forge clone that repo **read-only**, without storing a personal token or GitHub password on the server.
- After bootstrap, each developer still uses **their own GitHub SSH key** and permissions to push commits from inside the container.

This guide shows how to:

- let Forge clone **private** GitHub repos using per-dev deploy keys, and
- let each developer push commits using **their own** GitHub account.

## 1. Admin: configure a deploy key for a repo

Follow these steps once per `(developer, project)` when using `--use-deploy-key`.

1. Provision the dev environment:

   ```bash
   devctl add-dev --use-deploy-key
   ```

2. After running, `devctl` will:

   - create a deploy keypair under  
     `/opt/data/dev_workspaces/_deploy_keys/<project>/<dev>/id_ed25519(.pub)`
   - print the **public key** and target repo URL.

3. In GitHub, as the repo owner:

   - Open the repository.
   - Go to **Settings → Deploy keys**.
   - Click **“Add deploy key”**.
   - **Title**: something like `forge-<project>-<dev>`.
   - **Key**: paste the public key from step 2  
     (or read it from `id_ed25519.pub` on the VPS).
   - **Allow write access**: **leave this unchecked** (read-only).
   - Save.

4. Re-run the clone if needed:

   - If the initial bootstrap failed because the key wasn’t added yet, either:
     - re-run `devctl add-dev --use-deploy-key`, or
     - inside the container, run `git clone` / `git pull` again in `/workspace/<project>`.

## 2. Developer: connect the container to your GitHub account

Each developer should push with their **own** GitHub identity from inside the dev container.

In the dev container (as user `dev`):

1. Generate an SSH key (or copy an existing one):

   ```bash
   ssh-keygen -t ed25519 -C "your-email@example.com"
   # Accept the default path: /home/dev/.ssh/id_ed25519
   ```

2. Show the public key:

   ```bash
   cat ~/.ssh/id_ed25519.pub
   ```

3. Add this key to your GitHub account:

   - Go to GitHub → your avatar (top right) → **Settings**.
   - In the sidebar, choose **SSH and GPG keys**.
   - Click **“New SSH key”**, give it a name (e.g. `forge-dev-container`), paste the public key, and save.

4. Test SSH access from the container:

   ```bash
   ssh -T git@github.com
   ```

   You should see a message from GitHub confirming your identity.

5. Configure your Git identity inside the container:

   ```bash
   git config --global user.name "Your Name"
   git config --global user.email "your-email@example.com"
   ```

6. Work as usual in the project:

   ```bash
   cd /workspace/<project>
   git status
   git commit -m "Your change"
   git push
   ```

## 3. Summary of responsibilities

- **Forge (deploy keys)**:
  - creates per-dev+project deploy keys under `_deploy_keys/`
  - uses them to perform **read-only** `git clone` during `devctl add-dev --use-deploy-key`
  - never uses deploy keys for `git push`

- **Admin**:
  - adds deploy key public values as **read-only Deploy keys** on the GitHub repo
  - grants developers write access to the repo (collaborator/team/organization role)

- **Developer**:
  - configures their own SSH key inside the container and adds it to their GitHub account
  - pushes commits from the container using their own identity

