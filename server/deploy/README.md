# Server deployment

The server runs on a VPS under systemd and deploys itself: a timer polls this
repo's GitHub Releases, and when [server-deploy.yaml] has published a newer
`server-*` release, [paperdoll-deploy.sh] verifies and deploys it. See
[DEPLOY.md] for how a release gets cut.

[server-deploy.yaml]: ../../.github/workflows/server-deploy.yaml
[paperdoll-deploy.sh]: paperdoll-deploy.sh
[DEPLOY.md]: ../../DEPLOY.md

## Units

| Unit                          | Role                                                                        |
| ----------------------------- | --------------------------------------------------------------------------- |
| `paperdoll.target`            | Groups the two units below; a deploy restarts this.                         |
| `paperdoll-api.service`       | `app migrate up`, then `app serve`.                                         |
| `paperdoll-scheduler.service` | `app schedule`, started after the API server's migrations.                  |
| `paperdoll-deploy.timer`      | Starts `paperdoll-deploy.service` every 5 minutes.                          |
| `paperdoll-deploy.service`    | Runs `paperdoll-deploy.sh` as root. Logs: `journalctl -u paperdoll-deploy`. |

## Layout on the VPS

```
/etc/paperdoll/env                 DB_DSN and APP_* variables (see server/cfg)
/opt/paperdoll/releases/<version>/ app, app.sigstore.json
/opt/paperdoll/current             symlink to the deployed release
/var/lib/paperdoll-deploy/         release list cache, lock, failed/<version>
```

## Trust model

The VPS holds no GitHub credentials and GitHub holds none for the VPS. Before
deploying, the script checks the binary's Sigstore bundle with
`gh attestation verify`, requiring that it was built by `server-deploy.yaml` in
this repo, from the `server-<version>` tag, on a GitHub-hosted runner. Only the
release bot can push `server-*` tags (see Tag protection in [DEPLOY.md]), so a
binary passes only if it came out of the normal release pipeline. Releases older
than the deployed one are never deployed.

Verification needs outbound HTTPS to the Sigstore trust roots
(`tuf-repo-cdn.sigstore.dev`, `tuf-repo.github.com`) besides `api.github.com`
and `github.com`.

## Setting up the VPS

Requires bash, curl, jq, flock (util-linux), and [gh] >= 2.93.0. Run as root
from a checkout of this repo:

```sh
useradd --system --no-create-home --shell /usr/sbin/nologin paperdoll
install -d -m 0755 /opt/paperdoll/releases
install -d -m 0750 -g paperdoll /etc/paperdoll
install -m 0640 -g paperdoll /dev/null /etc/paperdoll/env  # then fill it in

# Stop and disable any units that ran the server before.

install -m 0755 server/deploy/paperdoll-deploy.sh /usr/local/bin/paperdoll-deploy
install -m 0644 server/deploy/paperdoll{.target,-*.service,-*.timer} /etc/systemd/system/
systemctl daemon-reload
systemctl enable paperdoll.target paperdoll-api.service paperdoll-scheduler.service
systemctl enable --now paperdoll-deploy.timer
```

The first timer run deploys the newest release and starts the server. To run it
right away: `systemctl start paperdoll-deploy.service`.

Nothing in `server/deploy/` updates itself: after changing any of these files,
repeat the `install` and `daemon-reload` steps above.

[gh]: https://github.com/cli/cli#installation

## When a deploy fails

- **Verification or download fails**: nothing changes on the VPS; the next run
  retries.
- **The new version fails its health check** (`/health` doesn't report it within
  60s, or a unit isn't active): the script switches back to the previous
  release, and creates `/var/lib/paperdoll-deploy/failed/<version>` so the timer
  stops retrying it. Delete that file to retry.

Migrations are not rolled back. A rollback starts the previous binary against
the newer schema, so migrations must stay backward-compatible with the release
before them.

## Rolling back by hand

```sh
cd /opt/paperdoll
touch /var/lib/paperdoll-deploy/failed/<bad-version>  # else the timer redeploys it
ln -sfn releases/<good-version> current.tmp && mv -T current.tmp current
systemctl restart paperdoll.target
```

Only the newest 5 releases are kept under `releases/`.
