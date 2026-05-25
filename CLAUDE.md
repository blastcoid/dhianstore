# Dhian Store

Monorepo untuk Dhian Store commerce services. GitHub: `blastcoid/dhianstore`.

Service-level files (go.mod, Dockerfile, README, dst) tinggal di subfolder service-nya, bukan di root.

## Planning workflow

1. Plan implementasi = **satu GitHub issue konsolidasi** (bukan dipecah per task). Phases dan sub-tasks pakai markdown checkbox di dalam issue.
2. Owner review issue dulu sebelum koding mulai.
3. Setelah owner approve, baru implementasi plannya di github issue

## Git workflow (GitHub Flow)

Single long-lived branch: `main`. Semua kerja via short-lived branch yang di-PR ke main, squash on merge, branch dihapus setelah merge.

Branch naming: `<type>/<issue-number>-<kebab-slug>`. Tipe:
- `feature/` — fitur baru / enhancement
- `fix/` — bug fix
- `hotfix/` — bug urgent di production
- `chore/` — maintenance (bump deps, rename, cleanup non-fungsional)
- `docs/`, `refactor/` — opsional, tambah saat butuh

PR body wajib include `Closes #<issue-number>` supaya issue auto-close saat merge ke main.

Hanya push ke main branch jika diminta oleh owner

## Repond rule
- Gunakan diagram mermaid saat menjelaskan tentang code atau arsitektur