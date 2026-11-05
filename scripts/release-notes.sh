#!/usr/bin/env bash
set -euo pipefail

tag="${1:-${GITHUB_REF_NAME:-}}"
if [[ -z "${tag}" ]]; then
	echo "usage: $0 <tag>" >&2
	exit 1
fi
repo="${GITHUB_REPOSITORY:-}"
if [[ -z "${repo}" ]]; then
	repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null || true)"
fi

previous="$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)"
commits="$(mktemp)"
trap 'rm -f "${commits}"' EXIT

if [[ -n "${previous}" ]]; then
	git log --no-merges "${previous}..${tag}" --pretty=format:'%s' > "${commits}"
else
	git log --no-merges "${tag}" --pretty=format:'%s' > "${commits}"
fi

echo "## ${tag} ($(date -u +%Y-%m-%d))"
if [[ -n "${previous}" && -n "${repo}" ]]; then
	echo
	echo "**Full Changelog**: https://github.com/${repo}/compare/${previous}...${tag}"
fi

print_section() {
	local title="$1"
	local list="$2"
	[[ -z "${list}" ]] && return 0
	echo
	echo "### ${title}"
	printf '%s\n' "${list}" | head -n 10 | sed 's/^/- /'
	local total
	total="$(printf '%s\n' "${list}" | wc -l | tr -d ' ')"
	if [[ "${total}" -gt 10 ]]; then
		echo "- … 等 $((total - 10)) 条"
	fi
}

print_section "新增" "$(grep -E '^feat(\(|:)' "${commits}" || true)"
print_section "修复" "$(grep -E '^fix(\(|:)' "${commits}" || true)"
print_section "重构" "$(grep -E '^refactor(\(|:)' "${commits}" || true)"
print_section "其他" "$(grep -Ev '^(feat(\(|:)|fix(\(|:)|refactor(\(|:))' "${commits}" || true)"
